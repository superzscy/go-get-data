package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func main() {
	target_url := "https://tps.ylbz.tj.gov.cn/jc/tps-local/b/#/addRequireSndl3Jx"
	wsURL := launcher.NewUserMode().Set("user-data-dir", "D:\\chrome_rod_usr_data").Leakless(false).MustLaunch()
	fmt.Println("wsURL:", wsURL)
	browser := rod.New().ControlURL(wsURL).MustConnect().NoDefaultDevice()

	page := browser.MustPage(target_url).MustWindowMaximize()

	fmt.Println("👉 请在打开的页面中完成登录，然后手动打开你想要的目标页面。完成后按回车继续。")
	fmt.Scanln() // 🔥 等你手动按回车继续

	// page.MustWaitStable().MustScreenshot("a.png")

	next_selector := "//*[@id=\"app\"]/div[1]/div[2]/section/div/div[2]/div[1]/div[2]/div/button[1]/i"

	loading_selector := "body > div.el-loading-mask.is-fullscreen.el-loading-fade-leave-active.el-loading-fade-leave-to"
	var str_rows []string

	for i := 1; i <= 68; i++ {
		fmt.Println("表格数据抓取中, 第", i, "页")
		newFunction(page, &str_rows)

		next_button := page.MustElementX(next_selector)
		next_button.MustClick()
		// 等待页面加载完成
		for page.MustHas(loading_selector) {
			fmt.Println("👉 页面正在加载，请稍等...")
			time.Sleep(1 * time.Second) // 等待1秒后再次检查
		}
	}

	// write to file csv
	file_name := "data.csv"
	file, err := os.Create(file_name)
	if err != nil {
		fmt.Println("🔥 创建文件失败:", err)
		return
	}

	writer := csv.NewWriter(file)

	// 写入表头
	writer.Write([]string{"品种名称;制剂规格;生产企业;历史中选药品;单位;2023年历史采购量;2024年历史采购量"})

	// 写入数据
	for _, row := range str_rows {
		writer.Write([]string{row})
	}
	writer.Flush()
	file.Close()

	// time.Sleep(time.Hour)
}

func newFunction(page *rod.Page, str_rows *[]string) {
	table_selector := "//*[@id=\"app\"]/div[1]/div[2]/section/div/div[2]/div[1]/div[1]/div[3]/table/tbody"

	table := page.MustElementX(table_selector)
	rows := table.MustElements("tbody > tr")
	// 遍历表格行
	for _, row := range rows {
		// 获取每一行的单元格
		cells := row.MustElements("td")
		row_str := ""
		// get the first 7 cells
		for i := 0; i < 7; i++ {
			row_str += cells[i].MustText() + ";"
		}
		*str_rows = append(*str_rows, row_str)
		fmt.Println("👉 获取到一行数据:", row_str)
	}
}
