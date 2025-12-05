-- name: GetCampaignByLinkID :one
SELECT * FROM campaigns
WHERE link_id = $1
LIMIT 1;