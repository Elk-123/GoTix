-- KEYS[1]: bitmap key
-- KEYS[2]: lock detail key
-- ARGV[1]: offset
-- ARGV[2]: user_id
-- ARGV[3]: ttl_seconds

local offset = tonumber(ARGV[1])
local ttl = tonumber(ARGV[3])

local is_taken = redis.call('GETBIT', KEYS[1], offset)

if is_taken == 1 then
    return 0
end

redis.call('SETBIT', KEYS[1], offset, 1)
redis.call('SET', KEYS[2], ARGV[2], 'EX', ttl)

return 1