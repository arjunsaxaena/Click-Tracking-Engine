import sys
import time
import requests
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict

DEFAULT_SERVER_URL = "http://localhost:4001"

BLOCKED_USER_AGENTS = [
    "curl/7.68.0",
    "wget/1.20.3",
    "python-requests/2.28.0",
]

NUM_REQUESTS = 105 # Expected behavious 100 redirects, 5 frauds


def make_request(
    server_url: str,
    link_id: str,
    user_id: str,
    user_agent: str,
    request_num: int,
    x_forwarded_for: str = None
) -> Dict:
    url = f"{server_url}/track/{link_id}"
    params = {
        "user_id": user_id,
        "gaid": f"test-gaid-{request_num}",
        "idfa": f"test-idfa-{request_num}",
    }
    headers = {
        "User-Agent": user_agent,
    }
    
    if x_forwarded_for:
        headers["X-Forwarded-For"] = x_forwarded_for
    
    try:
        start_time = time.time()
        response = requests.get(url, params=params, headers=headers, timeout=10)
        elapsed = time.time() - start_time
        
        return {
            "request_num": request_num,
            "status_code": response.status_code,
            "body": response.text[:200],
            "elapsed": elapsed,
            "success": True,
        }
    except Exception as e:
        return {
            "request_num": request_num,
            "status_code": None,
            "body": str(e),
            "elapsed": 0,
            "success": False,
            "error": str(e),
        }


def test_fraud_detection(server_url: str, link_id: str):
    print("=" * 80)
    print("FRAUD DETECTION TEST")
    print("=" * 80)
    print(f"Server URL: {server_url}")
    print(f"Link ID: {link_id}")
    print(f"Number of requests: {NUM_REQUESTS}")
    print(f"User-Agent: {BLOCKED_USER_AGENTS[0]} (blocked pattern)")
    print(f"IP Address: 192.168.1.100 (same IP for all requests)")
    print("=" * 80)
    print()
    
    user_agent = BLOCKED_USER_AGENTS[0]
    
    test_ip = "192.168.1.100"
    
    user_id_base = f"test-user-{int(time.time())}"
    
    print(f"Starting {NUM_REQUESTS} requests...")
    print("This will trigger:")
    print("  1. UA_BLOCKLIST: User-Agent matches blocked pattern")
    print("  2. IP_RATE_LIMIT: 100+ requests from same IP in 60 seconds")
    print()
    
    start_time = time.time()
    results: List[Dict] = []
    
    with ThreadPoolExecutor(max_workers=20) as executor:
        futures = []
        for i in range(1, NUM_REQUESTS + 1):
            user_id = f"{user_id_base}-{i}"
            future = executor.submit(
                make_request,
                server_url,
                link_id,
                user_id,
                user_agent,
                i,
                test_ip
            )
            futures.append(future)
        
        for future in as_completed(futures):
            result = future.result()
            results.append(result)
            if result["request_num"] % 10 == 0:
                print(f"  Completed {result['request_num']}/{NUM_REQUESTS} requests...")
    
    elapsed_time = time.time() - start_time
    
    results.sort(key=lambda x: x["request_num"])
    
    print()
    print("=" * 80)
    print("RESULTS SUMMARY")
    print("=" * 80)
    print(f"Total requests: {len(results)}")
    print(f"Total time: {elapsed_time:.2f} seconds")
    print(f"Requests per second: {len(results) / elapsed_time:.2f}")
    print()
    
    status_counts = {}
    fraud_responses = 0
    redirect_responses = 0
    error_responses = 0
    
    for result in results:
        if not result["success"]:
            error_responses += 1
            continue
        
        status = result["status_code"]
        status_counts[status] = status_counts.get(status, 0) + 1
        
        if status == 200 and "campaign not available" in result["body"].lower():
            fraud_responses += 1
        elif status == 302:
            redirect_responses += 1
    
    print("Status Code Distribution:")
    for status, count in sorted(status_counts.items()):
        print(f"  {status}: {count}")
    print()
    
    print("Response Analysis:")
    print(f"  Fraud responses (200 + 'campaign not available'): {fraud_responses}")
    print(f"  Redirect responses (302): {redirect_responses}")
    print(f"  Error responses: {error_responses}")
    print()
    
    print("Sample Responses:")
    print("-" * 80)
    
    print("\nFirst 3 responses:")
    for result in results[:3]:
        print(f"  Request #{result['request_num']}:")
        print(f"    Status: {result['status_code']}")
        print(f"    Body: {result['body'][:100]}...")
        print()
    
    if len(results) >= 100:
        print("\nResponses around request 100 (when IP rate limit should trigger):")
        for result in results[98:103]:
            print(f"  Request #{result['request_num']}:")
            print(f"    Status: {result['status_code']}")
            print(f"    Body: {result['body'][:100]}...")
            print()
    
    print("\nLast 3 responses:")
    for result in results[-3:]:
        print(f"  Request #{result['request_num']}:")
        print(f"    Status: {result['status_code']}")
        print(f"    Body: {result['body'][:100]}...")
        print()
    
    print("=" * 80)
    print("EXPECTED BEHAVIOR:")
    print("=" * 80)
    print("1. First ~99 requests: Should return 302 (redirect) - only UA_BLOCKLIST triggered")
    print("2. Request 100+: Should return 200 with 'campaign not available' - both")
    print("   UA_BLOCKLIST and IP_RATE_LIMIT triggered (2+ rules = FRAUD)")
    print()
    
    if fraud_responses > 0:
        print("✓ SUCCESS: Fraud detection is working!")
        print(f"  {fraud_responses} requests were marked as fraud")
    else:
        print("✗ WARNING: No fraud responses detected")
        print("  Check that:")
        print("    - User-Agent matches blocked pattern")
        print("    - 100+ requests were made within 60 seconds")
        print("    - All requests used the same IP address")
    
    print("=" * 80)


def main():
    if len(sys.argv) < 2:
        print("Usage: python test_fraud.py <link_id> [server_url]")
        print()
        print("Example:")
        print("  python test_fraud.py 123e4567-e89b-12d3-a456-426614174000")
        print("  python test_fraud.py 123e4567-e89b-12d3-a456-426614174000 http://localhost:4001")
        sys.exit(1)
    
    link_id = sys.argv[1]
    server_url = sys.argv[2] if len(sys.argv) > 2 else DEFAULT_SERVER_URL
    
    server_url = server_url.rstrip("/")
    
    print()
    print("Testing fraud detection...")
    print(f"Make sure your server is running at {server_url}")
    print(f"Make sure you have an active campaign with link_id: {link_id}")
    print()
    
    try:
        test_fraud_detection(server_url, link_id)
    except KeyboardInterrupt:
        print("\n\nTest interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n\nError: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()