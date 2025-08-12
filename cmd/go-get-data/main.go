package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

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

func isVisible(element *rod.Element) bool {
	style, _ := element.Attribute("style")
	if style != nil && strings.Contains(*style, "display:none") {
		return false
	}
	return true
}

func getTianJinData(browser *rod.Browser, skipNav bool) {
	var page *rod.Page

	if skipNav {
		pages, _ := browser.Pages()
		if len(pages) > 0 {
			page = pages[0]
			page.MustWindowMaximize()
		}
	} else {
		target_url := "https://tps.ylbz.tj.gov.cn/jc/tps-local/b/#/addRequireSndl3Jx"
		page = browser.MustPage()
		page.MustNavigate(target_url).MustWindowMaximize()
	}

	if page == nil {
		fmt.Println("创建页面失败:")
		return
	}

	fmt.Println("请在打开的页面中完成登录，然后手动打开你想要的目标页面。完成后按回车继续。")
	fmt.Scanln()

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
			fmt.Println("页面正在加载，请稍等...")
			time.Sleep(1 * time.Second) // 等待1秒后再次检查
		}
	}

	// write to file csv
	file_name := "data.csv"
	file, err := os.Create(file_name)
	if err != nil {
		fmt.Println("创建文件失败:", err)
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
}

func getSmpaaData(browser *rod.Browser, skipNav bool) {
	var page *rod.Page

	if skipNav {
		pages, _ := browser.Pages()
		if len(pages) > 0 {
			page = pages[0]
		}
	} else {
		page = browser.MustPage()
		target_url := "https://pub.smpaa.cn/login"
		page.MustNavigate(target_url).MustWindowMaximize()

		fmt.Println("请在打开的页面中完成登录，然后手动打开你想要的目标页面。完成后按回车继续。")
		fmt.Scanln()
	}

	iframeEl := page.MustElement("iframe#div0_1ItemFrame")
	iframe := iframeEl.MustFrame()

	table_xpath := "/html/body/div[2]/div[3]/div[3]/div[3]/div/table"

	main_table_row_cnt := 0

	if main_table_element, _ := iframe.ElementX(table_xpath); main_table_element != nil {
		if main_table_rows, _ := main_table_element.Elements("tr.ui-widget-content.jqgrow.ui-row-ltr"); main_table_rows != nil {
			main_table_row_cnt = len(main_table_rows)
		}
	}
	if main_table_row_cnt == 0 {
		fmt.Println("主表没有数据")
		return
	}

	fmt.Println("主表行数:", main_table_row_cnt)
	// var main_table [][]string
	var sub_table [][]string
	header_added := false

	for index_main := 0; index_main < main_table_row_cnt; index_main++ {
		// 由于会在主表和附表之间来回跳转, 需要重新获取表数据
		main_table_element, _ := iframe.ElementX(table_xpath)
		if main_table_element == nil {
			fmt.Println("主表元素未找到")
			return
		}
		main_table_rows, _ := main_table_element.Elements("tr.ui-widget-content.jqgrow.ui-row-ltr")
		if main_table_rows == nil {
			fmt.Println("主表元素未找到")
			return
		}
		main_table_row_cnt = len(main_table_rows)
		if main_table_row_cnt == 0 {
			fmt.Println("主表没有数据")
			return
		}
		if index_main >= main_table_row_cnt {
			fmt.Println("主表行数不足", index_main, main_table_row_cnt)
			return
		}
		fmt.Println("获取第", index_main, "行数据")

		row := main_table_rows[index_main]
		// 找当前行的所有单元格（td 或 th）
		// cells := row.MustElements("th, td")

		// var rowData []string
		// for _, cell := range cells {
		// 	// 去掉前后空格
		// 	text := strings.TrimSpace(cell.MustText())
		// 	rowData = append(rowData, text)
		// }
		// main_table = append(main_table, rowData)

		button_declare, _ := row.Element("a.a-declare.btn.btn-primary.btn-sm")
		if button_declare != nil {
			button_declare.MustClick()
			time.Sleep(1 * time.Second)

			if !header_added {
				header_added = true
				table_header_elem, _ := iframe.Element("table.ui-jqgrid-htable")
				if table_header_elem != nil {
					row := table_header_elem.MustElements("tr")[0]
					// 找当前行的所有单元格（td 或 th）
					cells := row.MustElements("th, td")
					var rowData []string
					for _, cell := range cells {
						style, _ := cell.Attribute("style")
						if style != nil && strings.Contains(*style, "display: none") {
							continue
						}
						// 去掉前后空格
						text := strings.TrimSpace(cell.MustText())
						// 去掉换行符
						text = strings.ReplaceAll(text, "\n", "")
						if len(rowData) == 0 && text == "" {
							text = "序号"
						}
						rowData = append(rowData, text)
					}
					// fmt.Println(strings.Join(rowData, ","))
					sub_table = append(sub_table, rowData)
				}
			}

			table_elem, _ := iframe.Element("table.els-jqGrid.ui-jqgrid-btable")
			if table_elem != nil {
				rows_declare := table_elem.MustElements("tr.ui-widget-content.jqgrow.ui-row-ltr")
				// fmt.Println("获取到的报量行数:", len(rows_declare))

				for _, row := range rows_declare {
					// 找当前行的所有单元格（td 或 th）
					cells := row.MustElements("th, td")
					var rowData []string
					for _, cell := range cells {
						style, _ := cell.Attribute("style")
						if style != nil && strings.Contains(*style, "display:none") {
							continue
						}

						// 去掉前后空格
						text := strings.TrimSpace(cell.MustText())
						rowData = append(rowData, text)
					}
					// fmt.Println(strings.Join(rowData, ","))
					sub_table = append(sub_table, rowData)
				}
			}

			button_back, _ := iframe.ElementX("/html/body/div[2]/form/div[2]/a[4]")
			if button_back != nil {
				button_back.MustClick()
				time.Sleep(1 * time.Second)
			}
		}
	}

	// 输出表格数据
	file, err := os.Create("output.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 假设有一行数据
	for _, row := range sub_table {
		if err := writer.Write(row); err != nil {
			log.Fatal(err)
		}
	}
}

type Command struct {
	Desc string
	Run  func(*rod.Browser, bool)
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("请选择操作:")
	fmt.Println(" 1 - 新开浏览器")
	fmt.Println(" 2 - 连接已有浏览器 (需已开启远程调试端口)")
	fmt.Print("输入: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	var browser *rod.Browser

	switch choice {
	case "1":
		// 新开一个 Chrome
		wsURL := launcher.NewUserMode().Set("user-data-dir", "D:\\chrome_rod_usr_data").Leakless(false).MustLaunch()
		fmt.Println("wsURL:", wsURL)
		browser = rod.New().ControlURL(wsURL).MustConnect().NoDefaultDevice()

	case "2":
		fmt.Print("请输入 WebSocket Debugger URL (例如 ws://127.0.0.1:9222/devtools/browser/xxxx): ")
		wsURL, _ := reader.ReadString('\n')
		wsURL = strings.TrimSpace(wsURL)

		if !strings.HasPrefix(wsURL, "ws://") {
			fmt.Println("❌ 无效的 WebSocket URL")
			return
		}

		browser = rod.New().ControlURL(wsURL).MustConnect()
		fmt.Println("✅ 已连接到已有浏览器")
	default:
		fmt.Println("❌ 无效选择")
		return
	}

	// 命令映射
	commands := map[string]Command{
		"1": {"获取天津数据", getTianJinData},
		"2": {"获取SMPAA数据", getSmpaaData},
	}

	reader = bufio.NewReader(os.Stdin)

	for name, cmd := range commands {
		fmt.Printf("  %-2s - %s\n", name, cmd.Desc)
	}
	fmt.Print("请输入: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if cmd, ok := commands[input]; ok {
		skipNav := choice == "2"

		cmd.Run(browser, skipNav)
	}
}
