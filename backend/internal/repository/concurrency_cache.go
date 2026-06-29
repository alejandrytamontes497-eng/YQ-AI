package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// 濡ょ姷鍋犲▔娑溿亹閸岀偛绠崇憸宥夊春濡ゅ啰纾介柟鎯х－閹界娀鎮介娑欏€愰柛锝堟閳ь剝顫夐惌顔剧不?//
// 闂佽鍎搁崱妤€骞嬫繛鏉戝悑閿氶悗浣冨皺閹风姷鈧稒蓱椤牠鏌?// 闂佸憡顭囬崰搴ㄦ偪閸曨垱鍋濆Λ棰佺閳诲繘鏌?SCAN 闂佸憡绋掗崹婵嬪箮閵堝鐒肩€广儱鎳庨崸濠囨煟濞嗘ê澧伴柣婵囩洴閹啴宕熼顐亝閹峰懎顓兼径瀣闂佹寧绋戝寮宯currency:account:{id}:{requestID}闂佹寧绋戦¨鈧紒?// 闂侀潻璐熼崝鎴︽偟椤曗偓閻涱喚鎹勯崫鍕矝闂侀潻濡囬崕銈呪枍濞嗘劗鈻?SCAN 闂傚倸娲犻崑鎾绘偡閺囨氨顦︽い锕€鐏濋埢搴ㄥ灳瀹曞洨鐛㈤柡澶嗘櫆濡垹妲愬┑鍥┾枖闁哄啫鐗呴幉楣冩煕濡儤顥滈柕鍥ф喘閺屽矁绠涘☉娆愭闂佸搫鍟抽崺鏍焵椤戭剙妫楅崢鏉戔槈閹炬剚鍎旀俊鍓у厴瀵増鎯旈姀鈾€鏋栭梺?//
// 闂佸搫鍊瑰姗€鎮块崟顖涘仢闁绘鐗婇弳顓㈡煟?Redis 闂佸搫鐗嗛ˇ顖滆姳椤撱垺鈷栭柛鈩冾殕閸娿倝鏌ㄥ☉妯绘睘orted Set闂佹寧绋戦¨鈧紒?// 1. 濠殿噯绲界换瀣煂濠婂懏瀚婚柨鏇楀亾鐟?闂佹椿娼块崝宥夊春濞戙垹鐭楁い蹇撴噺缁犳帒鈽夐幘顖氫壕婵炴垶鎼╂禍顏堝极椤撱垺鏅悘鐐村劤閻忓洭鏌涘☉娅偐鎷?requestID闂佹寧绋戦懟顖炲垂鎼淬劌鏋佸Λ棰佹祰缁€瀣煛閸愩劎鍩ｆ俊顐㈡健楠?// 2. 婵炶揪缍€濞夋洟寮?ZCARD 闂佸憡顭囬崰搴ㄦ偤濞嗘挻鍤旂€瑰嫭婢樼徊鎸庮殽閻愯埖纭剧憸鏉垮€垮顐ゆ暜椤斿墽顦梺鍝勫暙閻栫厧螞閹稿骸绶炵€广儱妫欑徊浠嬪箹?O(1)
// 3. 婵炶揪缍€濞夋洟寮?ZREMRANGEBYSCORE 濠电偞鎸搁幊鎰板箖婵犲啯浜ら柛銉ｅ妽閸╁倹淇婇崣澶婄亰缂傚秴鎳橀弫宥呯暆閸曨亞绱氶梺绋跨箰缁夊灚鏅跺鍫濈闁靛ě灞芥倎闂?TTL
// 4. Single Redis calls keep the hot path compact.
const (
	// 濡ょ姷鍋犲▔娑溿亹閸屾锝夊箣閹烘梻孝闂備焦顑欓崰鏍ㄦ櫠閻樼數纾介柍褜鍓熼弫宥夊醇閻斿摜鐣抽柟鍏兼綑缁绘ê鈻旈敃鍌氳Е闁割偒鍓涚粈?	// 闂佸搫绉堕崢褏妲? concurrency:account:{accountID}
	accountSlotKeyPrefix = "concurrency:account:"
	// 闂佸搫绉堕崢褏妲? concurrency:user:{userID}
	userSlotKeyPrefix = "concurrency:user:"
	// 缂備焦绋戦ˇ顖滄閻斿吋鈷撻柣鏂垮槻閻忔瑩鎮规担鎻掑⒉闁哄棛鍠栧畷鎶藉Ω閿旀儳顥曢悗? concurrency:wait:{userID}
	waitQueueKeyPrefix = "concurrency:wait:"
	// 闁荤姵鍔х粻鎴ｃ亹鐠恒劎妫憸蹇涙偤閹存繍鍤楅柛娑樼摠琚濋梺鍛婂笚椤ㄥ顢橀幖浣告瀬闁哄瀵ч悵銈夋煛瀹ュ洤甯剁紒? wait:account:{accountID}
	accountWaitKeyPrefix = "wait:account:"

	// 婵帗绋掗…鍫ヮ敇鐠囧弬锝夊箣閹烘梻孝闁哄鏅涘ú锕€锕㈤敓鐘茬睄闁割偅娲橀敍鐔兼煥濞戞澧曢柛銊ラ叄閺岋箓鎮ら崒婊咁槴闂佹寧绋戦懟顖濄亹閺屻儲鐒绘慨妯虹－缁犳牠姊洪弶璺ㄐら柣銈呮閹蹭即宕卞▎鎰紣
	defaultSlotTTLMinutes = 15
)

var (
	// acquireScript 婵炶揪缍€濞夋洟寮妶澶婂珘濠㈣泛锕︾喊宥夋⒒閸℃顥滈柟顔兼捣閹峰鍩勯崘鈺傤啀濡ょ姷鍋犲▔娑橈耿椤忓牆瀚夋い蹇撴处瑜把冣槈閹剧韬俊鐐そ瀵噣鎳滈棃娑欑秹闂佸憡姊绘慨浣冩綍婵?	// 婵炶揪缍€濞夋洟寮?Redis TIME 闂佸憡绋掗崹婵嬪箮閵堝鍤旂€瑰嫭婢樼徊鍧楁煛閸繄孝濠殿喚鍠栧畷鎶藉Ω閿旂瓔妲梻鍌氬€介幓顏嗘濠靛鐒奸柛顭戝枛鐢啿顭块懜浣冨闁汇倕瀚粭鐔衡偓锝庡亝椤ρ囨⒑閻ｅ苯鏋嶇紒妤€顦靛畷銉т沪缂併垹濡遍梻鍌氬亞閸ｏ綁銆?	// KEYS[1] = 闂佸搫鐗嗛ˇ顖滆姳椤撱垺鈷栭柛鈩冾殕閸娿倝姊?(concurrency:account:{id} / concurrency:user:{id})
	// ARGV[1] = maxConcurrency
	// ARGV[2] = TTL闂佹寧绋戦悧蹇涳綖濡ゅ懏鏅?	// ARGV[3] = requestID
	acquireScript = redis.NewScript(`
		local key = KEYS[1]
		local maxConcurrency = tonumber(ARGV[1])
		local ttl = tonumber(ARGV[2])
		local requestID = ARGV[3]
		local now = tonumber(ARGV[4])
		local expireBefore = now - ttl

		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)

		local exists = redis.call('ZSCORE', key, requestID)
		if exists ~= false then
			redis.call('ZADD', key, now, requestID)
			redis.call('EXPIRE', key, ttl)
			return 1
		end

		local count = redis.call('ZCARD', key)
		if count < maxConcurrency then
			redis.call('ZADD', key, now, requestID)
			redis.call('EXPIRE', key, ttl)
			return 1
		end

		return 0
	`)

	// getCountScript counts active slots after removing expired members.
	getCountScript = redis.NewScript(`
		local key = KEYS[1]
		local ttl = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local expireBefore = now - ttl

		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)
		return redis.call('ZCARD', key)
	`)

	// incrementWaitScript - refreshes TTL on each increment to keep queue depth accurate
	// KEYS[1] = wait queue key
	// ARGV[1] = maxWait
	// ARGV[2] = TTL in seconds
	incrementWaitScript = redis.NewScript(`
		local current = redis.call('GET', KEYS[1])
		if current == false then
			current = 0
		else
			current = tonumber(current)
		end

		if current >= tonumber(ARGV[1]) then
			return 0
		end

		local newVal = redis.call('INCR', KEYS[1])

		-- Refresh TTL so long-running traffic doesn't expire active queue counters.
		redis.call('EXPIRE', KEYS[1], ARGV[2])

			return 1
		`)

	// incrementAccountWaitScript - account-level wait queue count (refresh TTL on each increment)
	incrementAccountWaitScript = redis.NewScript(`
			local current = redis.call('GET', KEYS[1])
			if current == false then
				current = 0
			else
				current = tonumber(current)
			end

			if current >= tonumber(ARGV[1]) then
				return 0
			end

			local newVal = redis.call('INCR', KEYS[1])

			-- Refresh TTL so long-running traffic doesn't expire active queue counters.
			redis.call('EXPIRE', KEYS[1], ARGV[2])

			return 1
		`)

	// decrementWaitScript - same as before
	decrementWaitScript = redis.NewScript(`
			local current = redis.call('GET', KEYS[1])
			if current ~= false and tonumber(current) > 0 then
				redis.call('DECR', KEYS[1])
			end
			return 1
		`)

	// cleanupExpiredSlotsScript removes expired members from a slot set.
	cleanupExpiredSlotsScript = redis.NewScript(`
		local key = KEYS[1]
		local ttl = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local expireBefore = now - ttl
		redis.call('ZREMRANGEBYSCORE', key, '-inf', expireBefore)
		if redis.call('ZCARD', key) == 0 then
			redis.call('DEL', key)
		else
			redis.call('EXPIRE', key, ttl)
		end
		return 1
	`)

	// startupCleanupScript removes slots left by older server processes.
	startupCleanupScript = redis.NewScript(`
		local activePrefix = ARGV[1]
		local slotTTL = tonumber(ARGV[2])
		local removed = 0
		for i = 1, #KEYS do
			local key = KEYS[i]
			local members = redis.call('ZRANGE', key, 0, -1)
			for _, member in ipairs(members) do
				if string.sub(member, 1, string.len(activePrefix)) ~= activePrefix then
					removed = removed + redis.call('ZREM', key, member)
				end
			end
			if redis.call('ZCARD', key) == 0 then
				redis.call('DEL', key)
			else
				redis.call('EXPIRE', key, slotTTL)
			end
		end
		return removed
	`)
)

type concurrencyCache struct {
	rdb                 *redis.Client
	slotTTLSeconds      int
	waitQueueTTLSeconds int
}

// NewConcurrencyCache 闂佸憡甯楃粙鎴犵磽閹捐崵宓侀悹鍝勬惈缁叉椽鏌熺挩澶婂暙閻撴垹绱撻崒娑欏碍闁?// slotTTLMinutes: 濠碘€冲级閸ㄦ繄绱為崨顔戒氦闁搞儯鍔嶉崺鍌炴煛閸愩劎鍩ｆ俊顐㈡健閺佸秹宕煎┑鍡欌偓濠氭⒑閻ｅ苯娅愮紒杈ㄥ哺閺? 闂佺懓鐡ㄩ悧鐐电矆鐎ｎ喖鏋佸Λ棰佺閳诲繘鏌ｉ～顒€濮傜紒顕呭灣閹峰濡堕崨顏勪壕?15 闂佸憡甯掑Λ婵嬪箰?// waitQueueTTLSeconds: 缂備焦绋戦ˇ顖滄閻斿吋鈷撻柣鏂垮槻閻忔瑩寮堕埡浣瑰婵犫偓閿熺姴绫嶉柛顐ｆ礃閿涚喖鏌ㄥ☉妯煎ⅱ妞ゎ偅顨婇弫宥嗗緞濞戞氨顦? 闂佺懓鐡ㄩ悧鐐电矆鐎ｎ喖鏋佸Λ棰佺閳诲繘鏌?slot TTL
func NewConcurrencyCache(rdb *redis.Client, slotTTLMinutes int, waitQueueTTLSeconds int) service.ConcurrencyCache {
	if slotTTLMinutes <= 0 {
		slotTTLMinutes = defaultSlotTTLMinutes
	}
	if waitQueueTTLSeconds <= 0 {
		waitQueueTTLSeconds = slotTTLMinutes * 60
	}
	return &concurrencyCache{
		rdb:                 rdb,
		slotTTLSeconds:      slotTTLMinutes * 60,
		waitQueueTTLSeconds: waitQueueTTLSeconds,
	}
}

// Helper functions for key generation
func accountSlotKey(accountID int64) string {
	return fmt.Sprintf("%s%d", accountSlotKeyPrefix, accountID)
}

func userSlotKey(userID int64) string {
	return fmt.Sprintf("%s%d", userSlotKeyPrefix, userID)
}

func waitQueueKey(userID int64) string {
	return fmt.Sprintf("%s%d", waitQueueKeyPrefix, userID)
}

func accountWaitKey(accountID int64) string {
	return fmt.Sprintf("%s%d", accountWaitKeyPrefix, accountID)
}

// Account slot operations

func (c *concurrencyCache) AcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int, requestID string) (bool, error) {
	key := accountSlotKey(accountID)
	result, err := acquireScript.Run(ctx, c.rdb, []string{key}, maxConcurrency, c.slotTTLSeconds, requestID, time.Now().Unix()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *concurrencyCache) ReleaseAccountSlot(ctx context.Context, accountID int64, requestID string) error {
	key := accountSlotKey(accountID)
	return c.rdb.ZRem(ctx, key, requestID).Err()
}

func (c *concurrencyCache) GetAccountConcurrency(ctx context.Context, accountID int64) (int, error) {
	key := accountSlotKey(accountID)
	result, err := getCountScript.Run(ctx, c.rdb, []string{key}, c.slotTTLSeconds, time.Now().Unix()).Int()
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (c *concurrencyCache) GetAccountConcurrencyBatch(ctx context.Context, accountIDs []int64) (map[int64]int, error) {
	if len(accountIDs) == 0 {
		return map[int64]int{}, nil
	}

	now, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis TIME: %w", err)
	}
	cutoffTime := now.Unix() - int64(c.slotTTLSeconds)

	pipe := c.rdb.Pipeline()
	type accountCmd struct {
		accountID int64
		zcardCmd  *redis.IntCmd
	}
	cmds := make([]accountCmd, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		slotKey := accountSlotKeyPrefix + strconv.FormatInt(accountID, 10)
		pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		cmds = append(cmds, accountCmd{
			accountID: accountID,
			zcardCmd:  pipe.ZCard(ctx, slotKey),
		})
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	result := make(map[int64]int, len(accountIDs))
	for _, cmd := range cmds {
		result[cmd.accountID] = int(cmd.zcardCmd.Val())
	}
	return result, nil
}

// User slot operations

func (c *concurrencyCache) AcquireUserSlot(ctx context.Context, userID int64, maxConcurrency int, requestID string) (bool, error) {
	key := userSlotKey(userID)
	result, err := acquireScript.Run(ctx, c.rdb, []string{key}, maxConcurrency, c.slotTTLSeconds, requestID, time.Now().Unix()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *concurrencyCache) ReleaseUserSlot(ctx context.Context, userID int64, requestID string) error {
	key := userSlotKey(userID)
	return c.rdb.ZRem(ctx, key, requestID).Err()
}

func (c *concurrencyCache) GetUserConcurrency(ctx context.Context, userID int64) (int, error) {
	key := userSlotKey(userID)
	result, err := getCountScript.Run(ctx, c.rdb, []string{key}, c.slotTTLSeconds, time.Now().Unix()).Int()
	if err != nil {
		return 0, err
	}
	return result, nil
}

// Wait queue operations

func (c *concurrencyCache) IncrementWaitCount(ctx context.Context, userID int64, maxWait int) (bool, error) {
	key := waitQueueKey(userID)
	result, err := incrementWaitScript.Run(ctx, c.rdb, []string{key}, maxWait, c.waitQueueTTLSeconds).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *concurrencyCache) DecrementWaitCount(ctx context.Context, userID int64) error {
	key := waitQueueKey(userID)
	_, err := decrementWaitScript.Run(ctx, c.rdb, []string{key}).Result()
	return err
}

// Account wait queue operations

func (c *concurrencyCache) IncrementAccountWaitCount(ctx context.Context, accountID int64, maxWait int) (bool, error) {
	key := accountWaitKey(accountID)
	result, err := incrementAccountWaitScript.Run(ctx, c.rdb, []string{key}, maxWait, c.waitQueueTTLSeconds).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func (c *concurrencyCache) DecrementAccountWaitCount(ctx context.Context, accountID int64) error {
	key := accountWaitKey(accountID)
	_, err := decrementWaitScript.Run(ctx, c.rdb, []string{key}).Result()
	return err
}

func (c *concurrencyCache) GetAccountWaitingCount(ctx context.Context, accountID int64) (int, error) {
	key := accountWaitKey(accountID)
	val, err := c.rdb.Get(ctx, key).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return 0, err
	}
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return val, nil
}

func (c *concurrencyCache) GetAccountsLoadBatch(ctx context.Context, accounts []service.AccountWithConcurrency) (map[int64]*service.AccountLoadInfo, error) {
	if len(accounts) == 0 {
		return map[int64]*service.AccountLoadInfo{}, nil
	}

	now, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis TIME: %w", err)
	}
	cutoffTime := now.Unix() - int64(c.slotTTLSeconds)

	pipe := c.rdb.Pipeline()

	type accountCmds struct {
		id             int64
		maxConcurrency int
		zcardCmd       *redis.IntCmd
		getCmd         *redis.StringCmd
	}
	cmds := make([]accountCmds, 0, len(accounts))
	for _, acc := range accounts {
		slotKey := accountSlotKeyPrefix + strconv.FormatInt(acc.ID, 10)
		waitKey := accountWaitKeyPrefix + strconv.FormatInt(acc.ID, 10)
		pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		ac := accountCmds{
			id:             acc.ID,
			maxConcurrency: acc.MaxConcurrency,
			zcardCmd:       pipe.ZCard(ctx, slotKey),
			getCmd:         pipe.Get(ctx, waitKey),
		}
		cmds = append(cmds, ac)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	loadMap := make(map[int64]*service.AccountLoadInfo, len(accounts))
	for _, ac := range cmds {
		currentConcurrency := int(ac.zcardCmd.Val())
		waitingCount := 0
		if v, err := ac.getCmd.Int(); err == nil {
			waitingCount = v
		}
		loadRate := 0
		if ac.maxConcurrency > 0 {
			loadRate = (currentConcurrency + waitingCount) * 100 / ac.maxConcurrency
		}
		loadMap[ac.id] = &service.AccountLoadInfo{
			AccountID:          ac.id,
			CurrentConcurrency: currentConcurrency,
			WaitingCount:       waitingCount,
			LoadRate:           loadRate,
		}
	}

	return loadMap, nil
}

func (c *concurrencyCache) GetUsersLoadBatch(ctx context.Context, users []service.UserWithConcurrency) (map[int64]*service.UserLoadInfo, error) {
	if len(users) == 0 {
		return map[int64]*service.UserLoadInfo{}, nil
	}

	now, err := c.rdb.Time(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("redis TIME: %w", err)
	}
	cutoffTime := now.Unix() - int64(c.slotTTLSeconds)

	pipe := c.rdb.Pipeline()

	type userCmds struct {
		id             int64
		maxConcurrency int
		zcardCmd       *redis.IntCmd
		getCmd         *redis.StringCmd
	}
	cmds := make([]userCmds, 0, len(users))
	for _, u := range users {
		slotKey := userSlotKeyPrefix + strconv.FormatInt(u.ID, 10)
		waitKey := waitQueueKeyPrefix + strconv.FormatInt(u.ID, 10)
		pipe.ZRemRangeByScore(ctx, slotKey, "-inf", strconv.FormatInt(cutoffTime, 10))
		uc := userCmds{
			id:             u.ID,
			maxConcurrency: u.MaxConcurrency,
			zcardCmd:       pipe.ZCard(ctx, slotKey),
			getCmd:         pipe.Get(ctx, waitKey),
		}
		cmds = append(cmds, uc)
	}

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("pipeline exec: %w", err)
	}

	loadMap := make(map[int64]*service.UserLoadInfo, len(users))
	for _, uc := range cmds {
		currentConcurrency := int(uc.zcardCmd.Val())
		waitingCount := 0
		if v, err := uc.getCmd.Int(); err == nil {
			waitingCount = v
		}
		loadRate := 0
		if uc.maxConcurrency > 0 {
			loadRate = (currentConcurrency + waitingCount) * 100 / uc.maxConcurrency
		}
		loadMap[uc.id] = &service.UserLoadInfo{
			UserID:             uc.id,
			CurrentConcurrency: currentConcurrency,
			WaitingCount:       waitingCount,
			LoadRate:           loadRate,
		}
	}

	return loadMap, nil
}

func (c *concurrencyCache) CleanupExpiredAccountSlots(ctx context.Context, accountID int64) error {
	key := accountSlotKey(accountID)
	_, err := cleanupExpiredSlotsScript.Run(ctx, c.rdb, []string{key}, c.slotTTLSeconds, time.Now().Unix()).Result()
	return err
}

func (c *concurrencyCache) CleanupStaleProcessSlots(ctx context.Context, activeRequestPrefix string) error {
	if activeRequestPrefix == "" {
		return nil
	}

	slotPatterns := []string{accountSlotKeyPrefix + "*", userSlotKeyPrefix + "*"}
	for _, pattern := range slotPatterns {
		if err := c.cleanupSlotsByPattern(ctx, pattern, activeRequestPrefix); err != nil {
			return err
		}
	}

	waitPatterns := []string{accountWaitKeyPrefix + "*", waitQueueKeyPrefix + "*"}
	for _, pattern := range waitPatterns {
		if err := c.deleteKeysByPattern(ctx, pattern); err != nil {
			return err
		}
	}

	return nil
}

// cleanupSlotsByPattern scans slot keys and removes entries from old processes.
func (c *concurrencyCache) cleanupSlotsByPattern(ctx context.Context, pattern, activePrefix string) error {
	const scanCount = 200
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			return fmt.Errorf("scan %s: %w", pattern, err)
		}
		if len(keys) > 0 {
			_, err := startupCleanupScript.Run(ctx, c.rdb, keys, activePrefix, c.slotTTLSeconds).Result()
			if err != nil {
				return fmt.Errorf("cleanup slots %s: %w", pattern, err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// deleteKeysByPattern scans matching keys and deletes them in batches.
func (c *concurrencyCache) deleteKeysByPattern(ctx context.Context, pattern string) error {
	const scanCount = 200
	var cursor uint64
	for {
		keys, nextCursor, err := c.rdb.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			return fmt.Errorf("scan %s: %w", pattern, err)
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("del %s: %w", pattern, err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
