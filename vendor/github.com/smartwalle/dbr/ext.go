package dbr

import (
	"time"
)

func (this *Session) WithBlock(key string, blockValue string, blockSeconds int64) (bool, *Result) {
	return this.withBlock(key, blockValue, blockSeconds, 2, 500*time.Millisecond)
}

func (this *Session) withBlock(key string, blockValue string, blockSeconds int64, retryCount int, retryDelay time.Duration) (bool, *Result) {
	var rResult = this.GET(key)
	if rResult.Error != nil {
		return false, rResult
	}

	// 如果该 key 没有数据，则尝试对其进行执行写入操作
	if rResult.Data == nil {
		// 当从 redis 没有获取到数据的时候，写入 阻塞数据
		if this.SET(key, blockValue, "EX", blockSeconds, "NX").MustString() == "OK" {
			// 写入成功，直接返回不需要阻塞
			return false, nil
		}
		// 写入失败，延迟再次调用
		time.Sleep(time.Millisecond * 500)
		return this.WithBlock(key, blockValue, blockSeconds)
	}

	// 如果该 key 有数据，并且数据为 阻塞数据，则尝试延迟再查询，防止首先写入 阻塞数据 的业务未完成
	// 只能从一定程度上确保数据准确性
	if rResult.MustString() == blockValue {
		for i := 0; i < retryCount; i++ {
			time.Sleep(retryDelay)
			block, rResult2 := this.withBlock(key, blockValue, blockSeconds, 0, 0)
			if block == false {
				return false, rResult2
			}
		}

		// 该 key 有数据，并且数据等于 阻塞数据 的时候，返回阻塞
		return true, rResult
	}

	return false, rResult
}
