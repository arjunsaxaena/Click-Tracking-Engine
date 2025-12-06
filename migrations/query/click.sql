-- name: InsertClick :exec
INSERT INTO clicks (
    click_id, 
    timestamp, 
    link_id, 
    campaign_id, 
    user_id,
    ip_address,
    user_agent,
    referrer,
    device,
    device_model,
    browser,
    gaid,
    idfa,
    geo_country,
    geo_state,
    status,
    fraud_check_failed
) VALUES (
    $1,
    NOW(),
    $2, 
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14,
    $15,
    $16
);

-- name: CountClicksByIPInLast60Seconds :one
SELECT COUNT(*) as click_count
FROM clicks
WHERE ip_address = $1
  AND timestamp > NOW() - INTERVAL '60 seconds';