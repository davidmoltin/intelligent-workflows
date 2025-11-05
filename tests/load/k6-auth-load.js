// K6 load test for authentication endpoints
// Run with: k6 run k6-auth-load.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const loginAttempts = new Counter('login_attempts');
const successfulLogins = new Counter('successful_logins');
const tokenRefreshes = new Counter('token_refreshes');
const loginDuration = new Trend('login_duration');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Login load test
    login_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 },
        { duration: '1m', target: 50 },
        { duration: '30s', target: 0 },
      ],
      gracefulRampDown: '10s',
      exec: 'loginScenario',
    },
    // Scenario 2: Token refresh test
    token_refresh: {
      executor: 'constant-vus',
      vus: 10,
      duration: '2m',
      startTime: '30s',
      exec: 'tokenRefreshScenario',
    },
    // Scenario 3: API key usage
    api_key_usage: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      preAllocatedVUs: 20,
      stages: [
        { duration: '1m', target: 50 },
        { duration: '1m', target: 100 },
        { duration: '30s', target: 0 },
      ],
      exec: 'apiKeyScenario',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<300', 'p(99)<500'],
    'http_req_failed': ['rate<0.01'],
    'errors': ['rate<0.05'],
    'login_duration': ['p(95)<200'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Scenario 1: Login flow
export function loginScenario() {
  const username = `user-${__VU}-${Date.now()}`;
  const password = 'SecurePass123!';

  // Register new user
  const registerPayload = JSON.stringify({
    username: username,
    email: `${username}@example.com`,
    password: password,
  });

  const registerParams = {
    headers: { 'Content-Type': 'application/json' },
  };

  const registerRes = http.post(
    `${BASE_URL}/api/v1/auth/register`,
    registerPayload,
    registerParams
  );

  const registerCheck = check(registerRes, {
    'registration status is 201': (r) => r.status === 201,
    'registration returns tokens': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.refresh_token;
      } catch (e) {
        return false;
      }
    },
  });

  errorRate.add(!registerCheck);

  // Login with credentials
  const loginPayload = JSON.stringify({
    username: username,
    password: password,
  });

  loginAttempts.add(1);
  const loginStart = Date.now();

  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    loginPayload,
    registerParams
  );

  loginDuration.add(Date.now() - loginStart);

  const loginCheck = check(loginRes, {
    'login status is 200': (r) => r.status === 200,
    'login returns access token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (loginCheck) {
    successfulLogins.add(1);

    // Get user profile
    const tokens = JSON.parse(loginRes.body);
    const profileRes = http.get(`${BASE_URL}/api/v1/auth/me`, {
      headers: {
        'Authorization': `Bearer ${tokens.access_token}`,
      },
    });

    check(profileRes, {
      'profile status is 200': (r) => r.status === 200,
      'profile returns user data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.username === username;
        } catch (e) {
          return false;
        }
      },
    });
  } else {
    errorRate.add(1);
  }

  sleep(1);
}

// Scenario 2: Token refresh
export function tokenRefreshScenario() {
  // Register and login first
  const username = `refresh-user-${__VU}`;
  const password = 'SecurePass123!';

  const registerPayload = JSON.stringify({
    username: username,
    email: `${username}@example.com`,
    password: password,
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const registerRes = http.post(
    `${BASE_URL}/api/v1/auth/register`,
    registerPayload,
    params
  );

  if (registerRes.status === 201) {
    const tokens = JSON.parse(registerRes.body);

    // Refresh token
    const refreshPayload = JSON.stringify({
      refresh_token: tokens.refresh_token,
    });

    const refreshRes = http.post(
      `${BASE_URL}/api/v1/auth/refresh`,
      refreshPayload,
      params
    );

    const refreshCheck = check(refreshRes, {
      'refresh status is 200': (r) => r.status === 200,
      'refresh returns new tokens': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.access_token && body.refresh_token;
        } catch (e) {
          return false;
        }
      },
    });

    if (refreshCheck) {
      tokenRefreshes.add(1);
    } else {
      errorRate.add(1);
    }
  }

  sleep(2);
}

// Scenario 3: API key usage
export function apiKeyScenario() {
  // Create API key (simplified - in real scenario would use authenticated user)
  const apiKey = `test-api-key-${__VU}`;

  // Make authenticated request with API key
  const workflowRes = http.get(`${BASE_URL}/api/v1/workflows`, {
    headers: {
      'X-API-Key': apiKey,
    },
  });

  check(workflowRes, {
    'api key request completed': (r) => r.status === 200 || r.status === 401,
  });

  sleep(0.5);
}

// Test invalid login attempts
export function invalidLoginScenario() {
  const invalidPayload = JSON.stringify({
    username: 'nonexistent',
    password: 'wrongpassword',
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const loginRes = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    invalidPayload,
    params
  );

  check(loginRes, {
    'invalid login returns 401': (r) => r.status === 401,
  });

  sleep(1);
}

// Setup and teardown
export function setup() {
  console.log(`Starting authentication load test against ${BASE_URL}`);

  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    throw new Error(`Health check failed: ${healthRes.status}`);
  }

  console.log('Health check passed');
  return { baseUrl: BASE_URL };
}

export function teardown(data) {
  console.log('Authentication load test completed');
  console.log(`Total login attempts: ${loginAttempts.value}`);
  console.log(`Successful logins: ${successfulLogins.value}`);
  console.log(`Token refreshes: ${tokenRefreshes.value}`);
}
