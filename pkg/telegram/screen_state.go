package telegram

import "fmt"

const screenMessageCacheKeyFormat = "tg_screen_message:%d"

func ScreenMessageCacheKey(chatID int64) string {
	return fmt.Sprintf(screenMessageCacheKeyFormat, chatID)
}
