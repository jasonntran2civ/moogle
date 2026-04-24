// k6 load test against the search endpoint.
// Targets spec section 15.1 capacity: 5 QPS sustained, 50 burst.

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

const GATEWAY = __ENV.GATEWAY_URL || 'http://localhost:8080';

const QUERIES = [
  'sglt2 inhibitors heart failure',
  'crispr cas9 sickle cell',
  'glp-1 receptor agonist obesity',
  'covid-19 long term outcomes',
  'anticoagulation atrial fibrillation',
  'pembrolizumab non-small cell lung cancer',
  'metformin cardiovascular outcomes',
  'transcatheter aortic valve replacement',
];

const firstWaveLatency = new Trend('first_wave_ms', true);

export const options = {
  scenarios: {
    sustained: {
      executor: 'constant-arrival-rate',
      rate: 5,
      timeUnit: '1s',
      duration: __ENV.DURATION || '5m',
      preAllocatedVUs: 10,
      maxVUs: 60,
    },
  },
  thresholds: {
    'http_req_failed':                 ['rate<0.005'],
    'http_req_duration{name:search}':  ['p(95)<800'],
    'first_wave_ms':                   ['p(95)<250'],
  },
};

export default function () {
  const q = QUERIES[Math.floor(Math.random() * QUERIES.length)];
  const t0 = Date.now();
  const res = http.get(`${GATEWAY}/api/search?q=${encodeURIComponent(q)}&top_k=20`, {
    tags: { name: 'search' },
  });
  firstWaveLatency.add(Date.now() - t0);
  check(res, {
    'status 200': r => r.status === 200,
    'has results': r => r.json('results.length') > 0,
  });
  sleep(0.1);
}
