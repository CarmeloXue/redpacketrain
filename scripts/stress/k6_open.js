import http from 'k6/http';
import { check } from 'k6';

const campaigns = [1,2];
const baseURL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  scenarios: {
    open_red_packets: {
      executor: 'constant-arrival-rate',
      duration: '2m',
      rate: 5000,
      timeUnit: '1s',
      preAllocatedVUs: 1500,
      maxVUs: 6000,
    },
  },
};

function nextUserID() {
  const duplicateChance = Math.random();
  if (duplicateChance < 0.2) {
    return `user-repeat-${Math.floor(Math.random() * 1000)}`;
  }
  return `user-${Date.now()}-${__VU}-${__ITER}`;
}

export default function () {
  const campaignID = campaigns[Math.floor(Math.random() * campaigns.length)];
  const payload = JSON.stringify({ user_id: nextUserID() });
  const res = http.post(`${baseURL}/campaign/${campaignID}/open`, payload, {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '60s',
  });

  check(res, {
    'status acceptable': (r) => [200, 400, 404, 409, 410].includes(r.status),
  });
}
