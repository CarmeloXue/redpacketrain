local opened_key = KEYS[1]
local window_key = KEYS[2]
local amounts_key = KEYS[3]

local user_id = ARGV[1]
local now = tonumber(ARGV[2]) or tonumber(redis.call('TIME')[1])
local campaign_id = ARGV[3]

-- checking user eligibility
if redis.call('SISMEMBER', opened_key, user_id) == 1 then
    return {'ALREADY_OPENED', 0}
end

-- checking campaign availability
local window = redis.call('HMGET', window_key, 'start', 'end')
if window[1] == false or window[2] == false then
    return {'CAMPAIGN_NOT_FOUND', 0}
end

local start_ts = tonumber(window[1])
local end_ts = tonumber(window[2])
if not start_ts or not end_ts then
    return {'CAMPAIGN_NOT_FOUND', 0}
end

if now < start_ts or now > end_ts then
    return {'CAMPAIGN_INACTIVE', 0}
end

math.randomseed(now)


local amounts = redis.call('SMEMBERS', amounts_key)
if #amounts == 0 then
    return {'SOLD_OUT', 0}
end

while #amounts > 0 do
    local idx = math.random(#amounts)
    local amount = amounts[idx]
    amounts[idx] = amounts[#amounts]
    amounts[#amounts] = nil

    local inv_key = 'campaign:' .. campaign_id .. ':inv:' .. amount
    local remaining = tonumber(redis.call('GET', inv_key) or '0')
    if remaining > 0 then
        local new_count = redis.call('DECR', inv_key)
        if new_count >= 0 then
            redis.call('SADD', opened_key, user_id)
            return {'OK', tonumber(amount)}
        else
            redis.call('INCR', inv_key)
        end
    end
end

return {'SOLD_OUT', 0}
