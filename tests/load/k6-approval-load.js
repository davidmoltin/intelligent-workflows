// K6 load test for approval workflows
// Run with: k6 run k6-approval-load.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const approvalsCreated = new Counter('approvals_created');
const approvalsApproved = new Counter('approvals_approved');
const approvalsRejected = new Counter('approvals_rejected');
const approvalLatency = new Trend('approval_latency');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Create approval requests
    create_approvals: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 10 },
        { duration: '1m', target: 30 },
        { duration: '1m', target: 30 },
        { duration: '30s', target: 0 },
      ],
      exec: 'createApprovalScenario',
    },
    // Scenario 2: Process approvals
    process_approvals: {
      executor: 'constant-vus',
      vus: 5,
      duration: '2m',
      startTime: '30s',
      exec: 'processApprovalScenario',
    },
    // Scenario 3: List and query approvals
    query_approvals: {
      executor: 'constant-arrival-rate',
      rate: 20,
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 10,
      exec: 'queryApprovalScenario',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<400', 'p(99)<600'],
    'http_req_failed': ['rate<0.01'],
    'errors': ['rate<0.05'],
    'approval_latency': ['p(95)<300'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Shared state for approval IDs (simplified)
let approvalIds = [];

// Scenario 1: Create approval requests
export function createApprovalScenario() {
  const approvalPayload = JSON.stringify({
    workflow_execution_id: `exec-${__VU}-${Date.now()}`,
    approver_role: 'manager',
    context: {
      reason: 'High value transaction',
      amount: Math.random() * 10000 + 1000,
      customer_id: `customer-${__VU}`,
      risk_score: Math.random() * 100,
    },
    expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const createStart = Date.now();

  const createRes = http.post(
    `${BASE_URL}/api/v1/approvals`,
    approvalPayload,
    params
  );

  approvalLatency.add(Date.now() - createStart);

  const createCheck = check(createRes, {
    'create approval status is 201': (r) => r.status === 201,
    'create approval returns id': (r) => {
      try {
        const body = JSON.parse(r.body);
        if (body.id) {
          approvalIds.push(body.id);
          return true;
        }
        return false;
      } catch (e) {
        return false;
      }
    },
  });

  if (createCheck) {
    approvalsCreated.add(1);
  } else {
    errorRate.add(1);
  }

  sleep(1);
}

// Scenario 2: Process approvals (approve/reject)
export function processApprovalScenario() {
  // Create an approval first
  const approvalPayload = JSON.stringify({
    workflow_execution_id: `exec-process-${__VU}-${Date.now()}`,
    approver_role: 'manager',
    context: {
      reason: 'Process test',
      amount: Math.random() * 5000 + 500,
    },
    expires_at: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const createRes = http.post(
    `${BASE_URL}/api/v1/approvals`,
    approvalPayload,
    params
  );

  if (createRes.status === 201) {
    const approval = JSON.parse(createRes.body);
    const approvalId = approval.id;

    // Randomly approve or reject
    const shouldApprove = Math.random() > 0.3; // 70% approval rate

    const decisionPayload = JSON.stringify({
      decision: shouldApprove ? 'approved' : 'rejected',
      comment: shouldApprove
        ? 'Approved after review'
        : 'Rejected due to policy violation',
    });

    const endpoint = shouldApprove ? 'approve' : 'reject';
    const decisionRes = http.post(
      `${BASE_URL}/api/v1/approvals/${approvalId}/${endpoint}`,
      decisionPayload,
      params
    );

    const decisionCheck = check(decisionRes, {
      'decision status is 200': (r) => r.status === 200,
      'decision status matches': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.status === (shouldApprove ? 'approved' : 'rejected');
        } catch (e) {
          return false;
        }
      },
    });

    if (decisionCheck) {
      if (shouldApprove) {
        approvalsApproved.add(1);
      } else {
        approvalsRejected.add(1);
      }
    } else {
      errorRate.add(1);
    }
  }

  sleep(2);
}

// Scenario 3: Query approvals
export function queryApprovalScenario() {
  // List all approvals
  const listRes = http.get(`${BASE_URL}/api/v1/approvals`);

  const listCheck = check(listRes, {
    'list approvals status is 200': (r) => r.status === 200,
    'list returns array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body);
      } catch (e) {
        return false;
      }
    },
  });

  if (!listCheck) {
    errorRate.add(1);
  }

  // Query with filters (if supported)
  const filterRes = http.get(
    `${BASE_URL}/api/v1/approvals?status=pending&role=manager`
  );

  check(filterRes, {
    'filtered query status is 200': (r) => r.status === 200,
  });

  sleep(0.5);
}

// Scenario 4: Approval workflow with wait steps
export function approvalWorkflowScenario() {
  // Create workflow with approval step
  const workflowPayload = JSON.stringify({
    workflow_id: `approval-workflow-${__VU}`,
    version: '1.0.0',
    name: 'Approval Workflow Test',
    definition: {
      trigger: {
        type: 'event',
        event: 'transaction.submitted',
      },
      steps: [
        {
          id: 'check_amount',
          type: 'condition',
          condition: {
            field: 'transaction.amount',
            operator: 'gte',
            value: 5000,
          },
          on_true: 'require_approval',
          on_false: 'auto_approve',
        },
        {
          id: 'require_approval',
          type: 'wait',
          wait: {
            for_event: 'approval.granted',
            timeout: '24h',
          },
          next: 'approved',
        },
        {
          id: 'approved',
          type: 'action',
          action: {
            action: 'allow',
          },
        },
        {
          id: 'auto_approve',
          type: 'action',
          action: {
            action: 'allow',
          },
        },
      ],
    },
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const workflowRes = http.post(
    `${BASE_URL}/api/v1/workflows`,
    workflowPayload,
    params
  );

  if (workflowRes.status === 201) {
    // Trigger workflow
    const eventPayload = JSON.stringify({
      event_type: 'transaction.submitted',
      source: 'load-test',
      payload: {
        transaction_id: `txn-${__VU}-${Date.now()}`,
        amount: Math.random() * 10000 + 3000,
        customer_id: `customer-${__VU}`,
      },
    });

    const eventRes = http.post(
      `${BASE_URL}/api/v1/events`,
      eventPayload,
      params
    );

    check(eventRes, {
      'event trigger status is 202': (r) => r.status === 202,
    });
  }

  sleep(1);
}

// Setup and teardown
export function setup() {
  console.log(`Starting approval load test against ${BASE_URL}`);

  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    throw new Error(`Health check failed: ${healthRes.status}`);
  }

  console.log('Health check passed');
  return { baseUrl: BASE_URL };
}

export function teardown(data) {
  console.log('Approval load test completed');
  console.log(`Approvals created: ${approvalsCreated.value}`);
  console.log(`Approvals approved: ${approvalsApproved.value}`);
  console.log(`Approvals rejected: ${approvalsRejected.value}`);

  const totalProcessed = approvalsApproved.value + approvalsRejected.value;
  if (totalProcessed > 0) {
    const approvalRate = (approvalsApproved.value / totalProcessed * 100).toFixed(2);
    console.log(`Approval rate: ${approvalRate}%`);
  }
}
