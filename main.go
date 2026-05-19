package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// 使用 bufio.Reader 确保可以完整读取带空格的路径或字符串
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n==============================================")
		fmt.Println("                  文件批量提取器                ")
		fmt.Println("==============================================")

		// 1. 输入源目录
		fmt.Print("1. 请输入来源目录 (多个目录请用逗号 [,] 分隔):\n-> ")
		srcInput, _ := reader.ReadString('\n')
		srcInput = strings.TrimSpace(srcInput)
		if srcInput == "" {
			fmt.Println("[错误] 来源目录不能为空，请重新开始。")
			continue
		}

		// 2. 输入目标后缀
		fmt.Print("2. 请输入要提取的后缀名 (例如 .mp4 或 .mp4,.mkv , 留空默认 .mp4):\n-> ")
		extInput, _ := reader.ReadString('\n')
		extInput = strings.TrimSpace(extInput)
		if extInput == "" {
			extInput = ".mp4"
		}

		// 3. 输入关键字
		fmt.Print("3. 请输入文件名必须包含的关键字 (不需要请直接回车留空):\n-> ")
		matchInput, _ := reader.ReadString('\n')
		matchInput = strings.TrimSpace(matchInput)

		// 4. 输入导出目录
		fmt.Print("4. 请输入导出目标目录:\n-> ")
		outInput, _ := reader.ReadString('\n')
		outInput = strings.TrimSpace(outInput)
		if outInput == "" {
			fmt.Println("[错误] 导出目录不能为空，请重新开始。")
			continue
		}

		// 参数解析准备
		srcDirs := strings.Split(srcInput, ",")
		outDir := filepath.Clean(outInput)

		rawExts := strings.Split(extInput, ",")
		extMap := make(map[string]bool)
		for _, e := range rawExts {
			extMap[strings.ToLower(strings.TrimSpace(e))] = true
		}

		// 创建目标目录
		if err := os.MkdirAll(outDir, 0755); err != nil {
			fmt.Printf("[错误] 创建目标目录失败: %v\n", err)
			continue
		}

		fmt.Println("\n正在执行提取，请稍候...")
		startTime := time.Now()
		totalCopied := 0
		totalFailed := 0

		// 5. 核心遍历与提取逻辑
		for _, srcDir := range srcDirs {
			srcDir = strings.TrimSpace(srcDir)
			if srcDir == "" {
				continue
			}

			// 检查源目录是否存在
			if _, err := os.Stat(srcDir); os.IsNotExist(err) {
				fmt.Printf("[跳过] 目录不存在: %s\n", srcDir)
				continue
			}

			_ = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil // 遇到权限等错误的子目录直接跳过，不中断整个任务
				}
				if d.IsDir() {
					return nil
				}

				filename := d.Name()
				fileExt := strings.ToLower(filepath.Ext(filename))

				// 后缀过滤
				if !extMap[fileExt] {
					return nil
				}

				// 关键字过滤
				if matchInput != "" && !strings.Contains(filename, matchInput) {
					return nil
				}

				// 执行复制
				destPath := generateDestPath(outDir, filename)
				if err := copyFile(path, destPath); err != nil {
					fmt.Printf("[失败] %s -> %v\n", filename, err)
					totalFailed++
				} else {
					fmt.Printf("[成功] %s\n", filename)
					totalCopied++
				}
				return nil
			})
		}

		// 6. 给出统计结果
		duration := time.Since(startTime).Round(time.Millisecond)
		fmt.Println("\n====== 任务执行完毕 ======")
		fmt.Printf("总计耗时: %v\n", duration)
		fmt.Printf("成功提取: %d 个文件\n", totalCopied)
		if totalFailed > 0 {
			fmt.Printf("提取失败: %d 个文件\n", totalFailed)
		}
		fmt.Println("==========================")

		// 7. 阻断并等待回车继续
		fmt.Print("\n[提示] 按 [回车键(Enter)] 可以继续下一次运行...")
		_, _ = reader.ReadString('\n')
	}
}

// copyFile 实现底层文件流复制
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// generateDestPath 检测同名冲突，防覆盖
func generateDestPath(outDir, filename string) string {
	dest := filepath.Join(outDir, filename)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}
	// 冲突则在文件名后追加 Unix 纳秒时间戳
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	newName := fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
	return filepath.Join(outDir, newName)
}
