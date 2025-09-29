package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/log"
	"blueplan-research-dev-langgraph22/loomi-go/internal/loomi/utils"
)

// ExampleUnitTests 单元测试示例
func ExampleUnitTests() []TestSuite {
	return []TestSuite{
		{
			Name: "文本处理工具测试",
			Tests: []TestCase{
				{
					Name: "测试文本清理",
					Function: func() error {
						textUtils := utils.NewTextUtils(nil)
						result := textUtils.CleanText("  Hello   World  ")
						if result != "Hello World" {
							return fmt.Errorf("文本清理失败，期望 'Hello World'，实际 '%s'", result)
						}
						return nil
					},
					Timeout: 5 * time.Second,
				},
				{
					Name: "测试关键词提取",
					Function: func() error {
						textUtils := utils.NewTextUtils(nil)
						keywords := textUtils.ExtractKeywords("hello world hello go", 2)
						if len(keywords) != 2 {
							return fmt.Errorf("关键词提取失败，期望2个关键词，实际%d个", len(keywords))
						}
						return nil
					},
					Timeout: 5 * time.Second,
				},
				{
					Name: "测试文本截断",
					Function: func() error {
						textUtils := utils.NewTextUtils(nil)
						result := textUtils.TruncateText("This is a very long text", 10)
						if len(result) > 13 { // 10 + "..."
							return fmt.Errorf("文本截断失败，期望长度不超过13，实际%d", len(result))
						}
						return nil
					},
					Timeout: 5 * time.Second,
				},
			},
			Setup: func() error {
				fmt.Println("设置文本处理工具测试环境")
				return nil
			},
			Teardown: func() error {
				fmt.Println("清理文本处理工具测试环境")
				return nil
			},
		},
		{
			Name: "Markdown处理测试",
			Tests: []TestCase{
				{
					Name: "测试Markdown兼容性",
					Function: func() error {
						markdownProcessor := utils.NewMarkdownProcessor(nil)
						result := markdownProcessor.EnsureCompatibility("**bold** *italic*")
						if result == "" {
							return fmt.Errorf("Markdown兼容性处理失败")
						}
						return nil
					},
					Timeout: 5 * time.Second,
				},
				{
					Name: "测试Markdown元素提取",
					Function: func() error {
						markdownProcessor := utils.NewMarkdownProcessor(nil)
						elements := markdownProcessor.ExtractMarkdownElements("# Title\n- Item 1\n- Item 2")
						if len(elements) == 0 {
							return fmt.Errorf("Markdown元素提取失败")
						}
						return nil
					},
					Timeout: 5 * time.Second,
				},
			},
			Setup: func() error {
				fmt.Println("设置Markdown处理测试环境")
				return nil
			},
			Teardown: func() error {
				fmt.Println("清理Markdown处理测试环境")
				return nil
			},
		},
		{
			Name: "流式标签解析器测试",
			Tests: []TestCase{
				{
					Name: "测试标签解析",
					Function: func() error {
						parser := utils.NewStreamingTagParser(nil)
						tags := parser.AddChunk("<think>Hello</think>")
						if len(tags) == 0 {
							return fmt.Errorf("标签解析失败")
						}
						return nil
					},
					Timeout: 5 * time.Second,
				},
				{
					Name: "测试缓冲区管理",
					Function: func() error {
						parser := utils.NewStreamingTagParser(nil)
						parser.AddChunk("Hello")
						parser.AddChunk(" World")
						buffer := parser.GetBuffer()
						if buffer != "Hello World" {
							return fmt.Errorf("缓冲区管理失败")
						}
						return nil
					},
					Timeout: 5 * time.Second,
				},
			},
			Setup: func() error {
				fmt.Println("设置流式标签解析器测试环境")
				return nil
			},
			Teardown: func() error {
				fmt.Println("清理流式标签解析器测试环境")
				return nil
			},
		},
	}
}

// ExampleIntegrationTests 集成测试示例
func ExampleIntegrationTests() []TestSuite {
	return []TestSuite{
		{
			Name: "上下文管理器集成测试",
			Tests: []TestCase{
				{
					Name: "测试上下文创建和管理",
					Function: func() error {
						// 这里需要实际的Redis管理器，在真实测试中会提供
						// contextManager := utils.NewLoomiContextManager(logger, redisManager)
						// 创建上下文、添加消息、获取上下文等操作
						return nil
					},
					Timeout: 10 * time.Second,
				},
			},
			Setup: func() error {
				fmt.Println("设置上下文管理器集成测试环境")
				return nil
			},
			Teardown: func() error {
				fmt.Println("清理上下文管理器集成测试环境")
				return nil
			},
		},
		{
			Name: "告警管理器集成测试",
			Tests: []TestCase{
				{
					Name: "测试告警发送",
					Function: func() error {
						// 这里需要实际的配置，在真实测试中会提供
						// alertManager := utils.NewAlertManager(logger, config)
						// 测试发送告警到不同渠道
						return nil
					},
					Timeout: 15 * time.Second,
				},
			},
			Setup: func() error {
				fmt.Println("设置告警管理器集成测试环境")
				return nil
			},
			Teardown: func() error {
				fmt.Println("清理告警管理器集成测试环境")
				return nil
			},
		},
	}
}

// ExamplePerformanceTests 性能测试示例
func ExamplePerformanceTests() []BenchmarkCase {
	return []BenchmarkCase{
		{
			Name: "文本处理性能测试",
			Function: func() {
				textUtils := utils.NewTextUtils(nil)
				text := "This is a test text for performance testing. " +
					"It contains multiple sentences and words. " +
					"We will test the performance of text processing operations."

				for i := 0; i < 1000; i++ {
					textUtils.CleanText(text)
					textUtils.ExtractKeywords(text, 10)
					textUtils.TruncateText(text, 50)
				}
			},
		},
		{
			Name: "Markdown处理性能测试",
			Function: func() {
				markdownProcessor := utils.NewMarkdownProcessor(nil)
				markdown := "# Title\n\n**Bold text** and *italic text*.\n\n- Item 1\n- Item 2\n- Item 3"

				for i := 0; i < 1000; i++ {
					markdownProcessor.EnsureCompatibility(markdown)
					markdownProcessor.ExtractMarkdownElements(markdown)
				}
			},
		},
		{
			Name: "流式解析器性能测试",
			Function: func() {
				parser := utils.NewStreamingTagParser(nil)
				chunks := []string{
					"<think>",
					"Hello ",
					"World",
					"</think>",
					"<Observe>",
					"Test",
					"</Observe>",
				}

				for i := 0; i < 1000; i++ {
					for _, chunk := range chunks {
						parser.AddChunk(chunk)
					}
					parser.Reset()
				}
			},
		},
	}
}

// ExampleStressTests 压力测试示例
func ExampleStressTests() []StressTestCase {
	return []StressTestCase{
		{
			Name:              "文本处理压力测试",
			ConcurrentUsers:   10,
			IterationsPerUser: 100,
			Duration:          30 * time.Second,
			Function: func() error {
				textUtils := utils.NewTextUtils(nil)
				text := "This is a stress test for text processing."
				textUtils.CleanText(text)
				textUtils.ExtractKeywords(text, 5)
				return nil
			},
		},
		{
			Name:              "上下文管理压力测试",
			ConcurrentUsers:   5,
			IterationsPerUser: 50,
			Duration:          60 * time.Second,
			Function: func() error {
				// 模拟上下文管理操作
				return nil
			},
		},
	}
}

// ExampleEndToEndTests 端到端测试示例
func ExampleEndToEndTests() []EndToEndTestCase {
	return []EndToEndTestCase{
		{
			Name: "完整文本处理流程测试",
			Function: func() error {
				// 模拟完整的文本处理流程
				textUtils := utils.NewTextUtils(nil)
				markdownProcessor := utils.NewMarkdownProcessor(nil)
				parser := utils.NewStreamingTagParser(nil)

				// 1. 清理文本
				text := "  **Hello**   *World*  "
				cleanedText := textUtils.CleanText(text)

				// 2. 处理Markdown
				processedText := markdownProcessor.EnsureCompatibility(cleanedText)

				// 3. 解析标签
				tags := parser.AddChunk(processedText)

				// 验证结果
				if len(tags) == 0 && processedText == "" {
					return fmt.Errorf("端到端测试失败")
				}

				return nil
			},
		},
		{
			Name: "告警系统端到端测试",
			Function: func() error {
				// 模拟完整的告警流程
				// 1. 创建告警
				// 2. 发送到不同渠道
				// 3. 验证发送状态
				return nil
			},
		},
	}
}

// RunExampleTests 运行示例测试
func RunExampleTests() {
	// 创建测试框架
	logger := log.NewLogger()
	testFramework := NewTestFramework(logger)
	ctx := context.Background()

	// 运行单元测试
	fmt.Println("=== 运行单元测试 ===")
	unitTests := ExampleUnitTests()
	summary, err := testFramework.RunUnitTests(ctx, unitTests)
	if err != nil {
		fmt.Printf("单元测试失败: %v\n", err)
	} else {
		fmt.Printf("单元测试完成: 总计%d，通过%d，失败%d\n",
			summary.TotalTests, summary.PassedTests, summary.FailedTests)
	}

	// 运行性能测试
	fmt.Println("\n=== 运行性能测试 ===")
	performanceTests := ExamplePerformanceTests()
	benchmarkSummary, err := testFramework.RunPerformanceTests(ctx, performanceTests)
	if err != nil {
		fmt.Printf("性能测试失败: %v\n", err)
	} else {
		fmt.Printf("性能测试完成: 总计%d个基准测试\n", benchmarkSummary.TotalBenchmarks)
	}

	// 运行压力测试
	fmt.Println("\n=== 运行压力测试 ===")
	stressTests := ExampleStressTests()
	stressSummary, err := testFramework.RunStressTests(ctx, stressTests)
	if err != nil {
		fmt.Printf("压力测试失败: %v\n", err)
	} else {
		fmt.Printf("压力测试完成: 总计%d个测试\n", stressSummary.TotalTests)
	}

	// 运行端到端测试
	fmt.Println("\n=== 运行端到端测试 ===")
	e2eTests := ExampleEndToEndTests()
	e2eSummary, err := testFramework.RunEndToEndTests(ctx, e2eTests)
	if err != nil {
		fmt.Printf("端到端测试失败: %v\n", err)
	} else {
		fmt.Printf("端到端测试完成: 总计%d个测试\n", e2eSummary.TotalTests)
	}
}

// TestExample 测试示例函数
func TestExample(t *testing.T) {
	// 创建断言工具
	assert := NewAssert()

	// 测试文本处理
	textUtils := utils.NewTextUtils(nil)
	result := textUtils.CleanText("  Hello   World  ")
	assert.Equal(t, "Hello World", result)

	// 测试关键词提取
	keywords := textUtils.ExtractKeywords("hello world hello go", 2)
	assert.Equal(t, 2, len(keywords))

	// 测试Markdown处理
	markdownProcessor := utils.NewMarkdownProcessor(nil)
	elements := markdownProcessor.ExtractMarkdownElements("# Title\n- Item 1")
	assert.NotNil(t, elements)

	// 测试流式解析器
	parser := utils.NewStreamingTagParser(nil)
	tags := parser.AddChunk("<think>Hello</think>")
	assert.Equal(t, 1, len(tags))
}

// BenchmarkExample 基准测试示例
func BenchmarkTextProcessing(b *testing.B) {
	textUtils := utils.NewTextUtils(nil)
	text := "This is a benchmark test for text processing operations."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		textUtils.CleanText(text)
		textUtils.ExtractKeywords(text, 10)
		textUtils.TruncateText(text, 30)
	}
}

func BenchmarkMarkdownProcessing(b *testing.B) {
	markdownProcessor := utils.NewMarkdownProcessor(nil)
	markdown := "# Title\n\n**Bold text** and *italic text*.\n\n- Item 1\n- Item 2"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		markdownProcessor.EnsureCompatibility(markdown)
		markdownProcessor.ExtractMarkdownElements(markdown)
	}
}

func BenchmarkStreamingParser(b *testing.B) {
	parser := utils.NewStreamingTagParser(nil)
	chunk := "<think>Hello World</think>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.AddChunk(chunk)
		parser.Reset()
	}
}
