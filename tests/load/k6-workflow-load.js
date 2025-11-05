// K6 load test for workflow API
// Run with: k6 run k6-workflow-load.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up to 10 users
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '2m', target: 50 },   // Stay at 50 users
    { duration: '30s', target: 100 }, // Spike to 100 users
    { duration: '1m', target: 100 },  // Stay at 100 users
    { duration: '30s', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    'http_req_failed': ['rate<0.01'],   // Error rate should be less than 1%
    'errors': ['rate<0.1'],             // Custom error rate
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
const workflows = [
  {
    workflow_id: `load-test-${__VU}-${Date.now()}`,
    version: '1.0.0',
    name: 'Load Test Workflow',
    definition: {
      trigger: {
        type: 'event',
        event: 'order.created'
      },
      steps: [
        {
          id: 'step1',
          type: 'condition',
          condition: {
            field: 'order.total',
            operator: 'gt',
            value: 1000
          },
          on_true: 'step2',
          on_false: 'step3'
        },
        {
          id: 'step2',
          type: 'action',
          action: {
            action: 'block',
            reason: 'High value order'
          }
        },
        {
          id: 'step3',
          type: 'action',
          action: {
            action: 'allow'
          }
        }
      ]
    }
  }
];

export default function () {
  // Test 1: Create workflow
  const createPayload = JSON.stringify(workflows[0]);
  const createParams = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const createRes = http.post(`${BASE_URL}/api/v1/workflows`, createPayload, createParams);

  const createCheck = check(createRes, {
    'create workflow status is 201': (r) => r.status === 201,
    'create workflow response has id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  errorRate.add(!createCheck);

  if (createRes.status === 201) {
    const workflow = JSON.parse(createRes.body);

    // Test 2: Get workflow
    const getRes = http.get(`${BASE_URL}/api/v1/workflows/${workflow.id}`);

    const getCheck = check(getRes, {
      'get workflow status is 200': (r) => r.status === 200,
      'get workflow returns correct id': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.id === workflow.id;
        } catch (e) {
          return false;
        }
      },
    });

    errorRate.add(!getCheck);

    // Test 3: List workflows
    const listRes = http.get(`${BASE_URL}/api/v1/workflows`);

    const listCheck = check(listRes, {
      'list workflows status is 200': (r) => r.status === 200,
      'list workflows returns array': (r) => {
        try {
          const body = JSON.parse(r.body);
          return Array.isArray(body);
        } catch (e) {
          return false;
        }
      },
    });

    errorRate.add(!listCheck);

    // Test 4: Trigger event
    const eventPayload = JSON.stringify({
      event_type: 'order.created',
      source: 'load-test',
      payload: {
        order_id: `order-${__VU}-${Date.now()}`,
        total: 1500,
        customer_id: `customer-${__VU}`
      }
    });

    const eventRes = http.post(`${BASE_URL}/api/v1/events`, eventPayload, createParams);

    const eventCheck = check(eventRes, {
      'trigger event status is 202': (r) => r.status === 202,
    });

    errorRate.add(!eventCheck);
  }

  // Think time
  sleep(1);
}

// Setup function - runs once at the start
export function setup() {
  console.log(`Starting load test against ${BASE_URL}`);

  // Health check
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    throw new Error(`Health check failed: ${healthRes.status}`);
  }

  console.log('Health check passed');
}

// Teardown function - runs once at the end
export function teardown(data) {
  console.log('Load test completed');
}
