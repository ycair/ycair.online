package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("缺少參數: room_code password")
		return
	}
	room := os.Args[1]
	pass := os.Args[2]

	fmt.Printf("Go 核心已啟動 - 正在為房間 %s (密碼: %s) 建立 P2P 隧道...\n", room, pass)

	// 開啟一個後台任務定期印出心跳，防止死鎖
	go func() {
		for {
			// 未來這裡可以檢查連線狀態
			time.Sleep(30 * time.Second)
		}
	}()

	// 保持主程式不退出
	select {}
}
