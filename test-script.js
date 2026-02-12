import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    discardResponseBodies: true, // Не храним тела ответов
    scenarios: {
        breaking_point: {
            executor: 'ramping-vus',
            stages: [
                { duration: '10s', target: 500 },
                { duration: '20s', target: 1000 },
                { duration: '20s', target: 1500 },
                { duration: '20s', target: 2000 },
                { duration: '20s', target: 2500 },
                { duration: '30s', target: 3000 },
            ],
        },
    },
};

export default function () {
    const res = http.get('http://test-echo-server:8080');
    check(res, {
        'status is 200': (r) => r.status === 200,
    });
}