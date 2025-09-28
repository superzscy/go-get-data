package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/xuri/excelize/v2"
)

type Command struct {
	Desc string
	Run  func(*rod.Browser, bool)
}

var reader *bufio.Reader
var sWsUrl string
var sFilePath string
var sMaxProductNum int = 0
var sIndex int = 0
var sSearchResultPath string = `table > tbody > tr[class="ui-widget-content jqgrow ui-row-ltr"]`

type Product struct {
	index         int     // "序号",
	code          string  // "药品编码",
	name          string  // "药品名称",
	maker         string  // "生产厂家",
	supplier      string  // "供货商",
	spec          string  // "规格",
	num           int     // "数量",
	unit          string  // "单位",
	price         float32 // "进价",
	approval_code string  // "批准文号",
	platformCode  string  // "平台产品编号",
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

func reportData(browser *rod.Browser, skipNav bool) {

	var page *rod.Page
	pages, _ := browser.Pages()
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page = browser.MustPage()
		fmt.Println("请手动打开你想要的目标页面。完成后按回车继续。")
		fmt.Scanln()
	}
	page.MustWindowMaximize()
	iframeMain := page.MustElement("iframe#mainframe").MustFrame()

	f1, err := excelize.OpenFile(sFilePath)
	if err != nil {
		log.Fatal("读取 file1 失败")
		return
	}

	sheet1 := "Sheet1"
	rowsA, _ := f1.GetRows(sheet1)
	rowStringArray := [][]string{}

	for i, row := range rowsA {
		// 第一行是表头
		if i == 0 {
			continue
		}

		if sIndex > 0 {
			if i == sIndex {
				rowStringArray = append(rowStringArray, row)
				break
			}
		} else {
			if sMaxProductNum > 0 && i > sMaxProductNum {
				break
			}
			rowStringArray = append(rowStringArray, row)
		}
	}
	fmt.Println("读取到的行数:", len(rowStringArray))
	// 需要一个变量来记录平均耗时
	var totalDuration time.Duration = 0

	for index, row := range rowStringArray {
		func(index int, row []string) {
			beginTime := time.Now()

			// 计算耗时
			defer func() {
				duration := time.Since(beginTime)
				totalDuration += duration

				averageDuration := totalDuration / time.Duration(index+1)
				fmt.Printf("平均每条耗时: %v, 预计剩余时间: %v\n", averageDuration, averageDuration*time.Duration(len(rowStringArray)))
			}()

			var product Product
			// assign values to product fields
			fmt.Sscanf(row[0], "%d", &product.index)
			product.code = row[1]
			product.name = row[2]
			product.maker = row[3]
			product.supplier = row[4]
			product.spec = row[5]
			// convert string to int
			fmt.Sscanf(row[6], "%d", &product.num)
			product.unit = row[7]
			fmt.Sscanf(row[8], "%f", &product.price)
			product.approval_code = row[9]
			product.platformCode = row[10]

			fmt.Printf("开始添加商品, 序号: %d, 药品名称: %s, 产品编号: %s, 采购数量: %d, 采购价格: %f. 供应商: %s\n", product.index, product.name, product.code, product.num, product.price, product.supplier)

			// 带量采购
			iframeMain.MustElementR("a", "带量采购").MustClick()
			iframeMain.MustWaitStable()

			iframeMain.MustElementR("button", "清空").MustClick()

			// 更多
			has, elem, err := iframeMain.Has("#searchForm > div:nth-child(4) > div.moreButton")
			if err == nil && has && elem.MustText() == "更多" {
				elem.MustClick()
			}

			iframeMain.MustElementX("//*[@id=\"goodsId\"]").MustInput(product.platformCode)
			iframeMain.MustElementX("//*[@id=\"search1\"]").MustClick()
			iframeMain.MustWaitStable()

			has, elem, err = iframeMain.Has(sSearchResultPath)
			if err != nil {
				fmt.Println("[带量采购] 搜索错误:", err)
				return
			}

			if has {
				fmt.Println("[带量采购] 搜索到结果")
				elem.MustElement(`td > input[name="buyNum"]`).MustInput(strconv.Itoa(product.num))
				priceText := trimPriceString(elem.MustElement(`td[aria-describedby="gridlist_contractPriceInfo"]`).MustText())
				fmt.Println("价格:", priceText)

				// elem.MustElement(`td > input[name="buyNum"]`).MustInput("100")
				// elem.MustElement(`td > input[name="buyNum"]`).MustInput("100")

				elem.MustElement(`td[aria-describedby="gridlist_cb"] > input[class="cbox"]`).MustClick()
				iframeMain.MustElementR("button", "加入订单").MustClick()

				return

			} else {
				fmt.Println("[带量采购] 未搜索到结果")
			}

			// 普通采购
			iframeMain.MustElementR("a", "普通采购").MustClick()
			iframeMain.MustWaitStable()

			iframeMain.MustElementX("//*[@id=\"clear\"]").MustClick()
			iframeMain.MustElementX("//*[@id=\"procurecatalogId\"]").MustInput(product.platformCode)
			iframeMain.MustElementX("//*[@id=\"search1\"]").MustClick()
			iframeMain.MustWaitStable()

			has, elem, err = iframeMain.Has(sSearchResultPath)
			if err != nil {
				fmt.Println("[普通采购] 搜索错误:", err)
				return
			}
			if has {
				fmt.Println("[普通采购] 搜索到结果")
				elem.MustElement(`td > input[name="buyNum"]`).MustInput(strconv.Itoa(product.num))
				priceText := trimPriceString(elem.MustElement(`td[aria-describedby="gridlist_contractPriceInfo"]`).MustText())
				fmt.Println("价格:", priceText)
				elem.MustElement(`td[aria-describedby="gridlist_cb"] > input[class="cbox"]`).MustClick()
				iframeMain.MustElementR("button", "加入订单").MustClick()
			} else {
				fmt.Println("[普通采购] 未搜索到结果")
			}
		}(index, row)
	}
}

func trimPriceString(priceText string) string {
	priceText = strings.TrimPrefix(priceText, "￥")
	priceText = strings.ReplaceAll(priceText, ",", "")
	priceText = strings.TrimRight(priceText, "0")
	return priceText
}

func main() {

	data, err := os.ReadFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		log.Fatal(err)
	}

	if pretty, err := json.MarshalIndent(m, "", "  "); err != nil {
		fmt.Println("json 格式化失败:", err)
	} else {
		fmt.Println(string(pretty))
	}
	if wsurl, ok := m["wsurl"].(string); ok {
		sWsUrl = wsurl
	}
	if filepath, ok := m["filepath"].(string); ok {
		sFilePath = filepath
	}
	if maxProductNum, ok := m["maxProductNum"].(float64); ok {
		sMaxProductNum = int(maxProductNum)
	}
	if index, ok := m["index"].(float64); ok {
		sIndex = int(index)
	}

	fmt.Println("请选择操作:")
	fmt.Println(" 1 - 新开浏览器")
	fmt.Println(" 2 - 连接已有浏览器 (需已开启远程调试端口)")
	fmt.Print("输入: ")

	reader = bufio.NewReader(os.Stdin)
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
		if wsURL == "" {
			wsURL = sWsUrl
		}

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
		"3": {"报量", reportData},
	}
	// sort the commands by key in ascending order
	var keys []string
	for k := range commands {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	reader = bufio.NewReader(os.Stdin)

	for _, k := range keys {
		cmd := commands[k]
		fmt.Printf("  %-2s - %s\n", k, cmd.Desc)
	}
	fmt.Print("请输入: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if cmd, ok := commands[input]; ok {
		skipNav := choice == "2"

		cmd.Run(browser, skipNav)
	}
}
