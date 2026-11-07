package shipments

import (
 "context"
 "crypto/sha256"
 "fmt"
 "time"
 "github.com/redis/go-redis/v9"
)

type CostRateLimiter interface { Allow(ctx context.Context, clientIP string) (bool,error) }
type RedisCostRateLimiter struct { client *redis.Client; limit int64; globalLimit int64; window time.Duration }

// The optional global limit caps total provider usage across all clients and replicas.
// Existing callers remain compatible; default global capacity equals the per-IP limit.
func NewRedisCostRateLimiter(client *redis.Client, limit int, window time.Duration, globalLimit ...int) *RedisCostRateLimiter {
 global := limit
 if len(globalLimit)>0 { global=globalLimit[0] }
 return &RedisCostRateLimiter{client:client,limit:int64(limit),globalLimit:int64(global),window:window}
}

var costLimitScript=redis.NewScript(`
local ipCount = redis.call('GET', KEYS[1])
local globalCount = redis.call('GET', KEYS[2])
if tonumber(ipCount or '0') >= tonumber(ARGV[2]) then return 0 end
if tonumber(globalCount or '0') >= tonumber(ARGV[3]) then return 0 end
local ipNew = redis.call('INCR', KEYS[1])
if ipNew == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
local globalNew = redis.call('INCR', KEYS[2])
if globalNew == 1 then redis.call('PEXPIRE', KEYS[2], ARGV[1]) end
return 1
`)

func (l *RedisCostRateLimiter) Allow(ctx context.Context, clientIP string) (bool,error) {
 if l==nil || l.client==nil || l.limit<1 || l.globalLimit<1 || l.window<=0 {return false,fmt.Errorf("shipping quote rate limiter is not configured")}
 key:=fmt.Sprintf("shipping:quote:rate:v2:ip:%x",sha256.Sum256([]byte(clientIP)))
 result,err:=costLimitScript.Run(ctx,l.client,[]string{key,"shipping:quote:rate:v2:global"},l.window.Milliseconds(),l.limit,l.globalLimit).Int64()
 if err!=nil{return false,err}
 return result==1,nil
}
