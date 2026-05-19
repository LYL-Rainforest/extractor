package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// 终端 ANSI 颜色常量
const (
	ColorBlue   = "\033[1;34m"
	ColorGreen  = "\033[1;32m"
	ColorWhite  = "\033[1;37m"
	ColorReset  = "\033[0m"
	ClearScreen = "\033[H\033[2J" // 物理清屏并复位光标
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		// 1. 物理清屏，确保每一次“重来”都是干净的头部界面
		fmt.Print(ClearScreen)

		// 2. 蓝色大标题及外框
		fmt.Println("\033[36m+------------------------------------------+\033[0m")
		fmt.Println("\033[36m|               文件提取工具               |\033[0m")
		fmt.Println("\033[36m+------------------------------------------+\033[0m")

		// 3. 输入源路径（支持多个文件夹/文件拖入）
		fmt.Printf("%s[1] 请拖入【单个/多个 文件夹】并按回车:%s\n-> ", ColorWhite, ColorReset)
		if !scanner.Scan() {
			break
		}
		rawPaths := scanner.Text()
		targets := parseDragPaths(rawPaths)
		if len(targets) == 0 {
			fmt.Println("未检测到有效路径，请重试。")
			waitForRestart(scanner)
			continue
		}

		// 4. 输入目标输出目录
		fmt.Printf("\n%s[2] 请拖入或输入【提取后存放】的目标文件夹路径:%s\n-> ", ColorWhite, ColorReset)
		if !scanner.Scan() {
			break
		}
		outDir := strings.Trim(scanner.Text(), ` "`)

		// 5. 输入后缀名规则（留空默认）
		fmt.Printf("\n%s[3] 请输入要提取的【后缀名】(留空默认不限后缀):%s\n-> ", ColorWhite, ColorReset)
		if !scanner.Scan() {
			break
		}
		extRule := strings.TrimSpace(scanner.Text())
		if extRule != "" && !strings.HasPrefix(extRule, ".") {
			extRule = "." + extRule
		}

		// 6. 输入关键字规则
		fmt.Printf("\n%s[4] 请输入要检索的【文件名关键字】(留空默认匹配所有):%s\n-> ", ColorWhite, ColorReset)
		if !scanner.Scan() {
			break
		}
		keyRule := strings.TrimSpace(scanner.Text())

		// 7. 执行物理扫描与提取工作
		fmt.Printf("\n%s正在建立检索索引，开始执行物理提取...%s\n", ColorBlue, ColorReset)
		executeExtract(targets, outDir, extRule, keyRule)

		// 8. 按下回车清空当前屏幕并重来（已修正为绿色提示）
		fmt.Printf("\n%s[ 任务结束 ] 按回车键 (Enter) 重新开始...%s\n", ColorGreen, ColorReset)
		waitForRestart(scanner)
	}
}

// 解析拖拽进终端的路径（智能处理双引号与空格隔离的多路径）
func parseDragPaths(raw string) []string {
	var paths []string
	var sb strings.Builder
	inQuotes := false

	// 遍历解析拖入的路径字符串，兼容物理拖拽的引号闭合机制
	for i := 0; i < len(raw); i++ {
		char := raw[i]
		if char == '"' {
			inQuotes = !inQuotes
			continue
		}
		if char == ' ' && !inQuotes {
			if sb.Len() > 0 {
				paths = append(paths, sb.String())
				sb.Reset()
			}
			continue
		}
		sb.WriteByte(char)
	}
	if sb.Len() > 0 {
		paths = append(paths, sb.String())
	}
	return paths
}

// 执行核心文件搜索与物理拷贝
func executeExtract(targets []string, outDir, extRule, keyRule string) {
	// 创建输出目录
	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		fmt.Printf("创建目标文件夹失败: %v\n", err)
		return
	}

	successCount := 0

	for _, target := range targets {
		info, err := os.Stat(target)
		if err != nil {
			fmt.Printf("路径无效，跳过: %s\n", target)
			continue
		}

		// 情况 A: 拖入的是单个文件
		if !info.IsDir() {
			if matchFile(target, extRule, keyRule) {
				if copyFile(target, filepath.Join(outDir, info.Name())) {
					successCount++
				}
			}
			continue
		}

		// 情况 B: 拖入的是文件夹，执行深度递归扫描
		err = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if matchFile(path, extRule, keyRule) {
				if copyFile(path, filepath.Join(outDir, d.Name())) {
					successCount++
				}
			}
			return nil
		})
		if err != nil {
			fmt.Printf("扫描文件夹 [%s] 出错: %v\n", target, err)
		}
	}

	fmt.Printf("\n%s提取完成！成功物理复制了 %d 个文件到目标目录。%s\n", ColorGreen, successCount, ColorReset)
}

// 核心过滤规则判断：后缀名与关键字
func matchFile(path, extRule, keyRule string) bool {
	filename := filepath.Base(path)

	// 1. 校验后缀名规则（如果留空，直接跳过此项检测，进入关键字匹配）
	if extRule != "" {
		if !strings.EqualFold(filepath.Ext(path), extRule) {
			return false
		}
	}

	// 2. 校验关键字规则
	if keyRule != "" {
		if !strings.Contains(strings.ToLower(filename), strings.ToLower(keyRule)) {
			return false
		}
	}

	return true
}

// 物理流拷贝文件（改版：重名三位数递增探测逻辑）
func copyFile(src, dst string) bool {
	ext := filepath.Ext(dst)
	base := strings.TrimSuffix(dst, ext)

	// 增量计数器探测物理空位
	counter := 1
	for {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			// 文件不存在，这个物理路径可用，跳出循环执行复制
			break
		}
		// 文件存在，生成带三位格式化数字的新路径（例如 _001, _002）再进行下一轮探测
		dst = fmt.Sprintf("%s_%03d%s", base, counter, ext)
		counter++
	}

	input, err := os.ReadFile(src)
	if err != nil {
		return false
	}
	err = os.WriteFile(dst, input, 0666)
	return err == nil
}

// 阻塞等待回车
func waitForRestart(scanner *bufio.Scanner) {
	if scanner.Scan() {
		// 仅作为阻塞，按下任意回车直接放行进入下一次主循环
	}
}
