-- Create default collections for all users who don't have one
-- This ensures every user has at least one "Saved" collection

-- Insert a "Saved" collection for each user that doesn't have a collection with ID 1
INSERT INTO collections (id, created_at, updated_at, deleted_at, user_id, name)
SELECT 
    user_id as id,  -- Use user_id as collection ID for consistency
    NOW() as created_at,
    NOW() as updated_at,
    NULL as deleted_at,
    user_id,
    'Saved' as name
FROM 
    (SELECT DISTINCT id as user_id FROM users) u
WHERE 
    NOT EXISTS (
        SELECT 1 FROM collections c WHERE c.user_id = u.user_id
    )
ON CONFLICT (id) DO NOTHING;

-- Alternative: Create collection with sequential IDs
-- Uncomment if you prefer sequential IDs instead of user-based IDs
/*
INSERT INTO collections (created_at, updated_at, deleted_at, user_id, name)
SELECT 
    NOW() as created_at,
    NOW() as updated_at,
    NULL as deleted_at,
    user_id,
    'Saved' as name
FROM 
    (SELECT DISTINCT id as user_id FROM users) u
WHERE 
    NOT EXISTS (
        SELECT 1 FROM collections c WHERE c.user_id = u.user_id
    );
*/

-- Verify the results
SELECT u.id as user_id, u.username, c.id as collection_id, c.name
FROM users u
LEFT JOIN collections c ON c.user_id = u.id
ORDER BY u.id;
