package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"github.com/eiannone/keyboard"
)

// CommandHandler обрабатывает команды
type CommandHandler struct {
	langData               *LanguageData
	config                 *KeyboardConfig
	layouts                *ParsedLayouts
	analyses               []LayoutAnalysis
	bestResults            []SimulatedAnnealingResult // Store search results
	invertedLayout         *Layout                    // Store inverted layout from inv command
	searchResultLayout     *Layout                    // Store the last search result layout (temporary layout [0])
	isInvertedLayoutActive bool                       // Flag to indicate if inverted layout should be displayed for index 0
	highlightedLayouts     map[int]bool               // Store highlighted layouts by number
	configTracker          *ConfigChangeTracker       // Track configuration changes
	langFile               string
	configFile             string
	layoutFile             string
	outputFile             string  // File where new layouts will be saved
	effortFile             string  // Optional file for effort matrix (if provided via --effort option)
}

// NewCommandHandler создаёт новый обработчик команд
func NewCommandHandler(langData *LanguageData, config *KeyboardConfig, layouts *ParsedLayouts, langFile, configFile, layoutFile, outputFile, effortFile string) *CommandHandler {
	return &CommandHandler{
		langData:               langData,
		config:                 config,
		layouts:                layouts,
		bestResults:            make([]SimulatedAnnealingResult, 0),
		invertedLayout:         nil,
		searchResultLayout:     nil,
		isInvertedLayoutActive: false,
		highlightedLayouts:     make(map[int]bool),
		configTracker:          NewConfigChangeTracker(config.Weights),
		langFile:               langFile,
		configFile:             configFile,
		layoutFile:             layoutFile,
		outputFile:             outputFile,
		effortFile:             effortFile,
	}
}

// parseIndexRanges парсит спецификацию индексов и диапазонов (например "2,5,7-9,11-15")
func parseIndexRanges(spec string, maxIndex int) ([]int, error) {
	var indices []int
	indexMap := make(map[int]bool)

	parts := strings.Split(strings.TrimSpace(spec), ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			// Это диапазон
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("неверный формат диапазона: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("неверное начало диапазона: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("неверный конец диапазона: %s", rangeParts[1])
			}

			for i := start; i <= end; i++ {
				if i >= 1 && i <= maxIndex {
					indexMap[i-1] = true // Переводим в 0-based индекс
				}
			}
		} else {
			// Это одиночный индекс
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("неверный индекс: %s", part)
			}

			if idx >= 1 && idx <= maxIndex {
				indexMap[idx-1] = true
			}
		}
	}

	for i := 0; i < maxIndex; i++ {
		if indexMap[i] {
			indices = append(indices, i)
		}
	}
	sort.Ints(indices)

	return indices, nil
}

// getLayoutByIndex gets layout by 1-based index, where 0 is the search result layout
func (ch *CommandHandler) getLayoutByIndex(index int) (*Layout, bool) {
	// If index is 0 and inverted layout is active, return the inverted layout
	if index == 0 && ch.isInvertedLayoutActive && ch.invertedLayout != nil {
		return ch.invertedLayout, true
	}

	// If index is 0 and inverted layout is not active, return the search result layout (temporary layout [0])
	if index == 0 && ch.searchResultLayout != nil {
		return ch.searchResultLayout, true
	}

	// If index is 1 or greater, return from the loaded layouts
	if index > 0 && index <= len(ch.layouts.Layouts) {
		return &ch.layouts.Layouts[index-1], true // Convert to 0-based index
	}

	return nil, false
}

// getLayoutCount returns the total number of layouts including temporary ones
func (ch *CommandHandler) getLayoutCount() int {
	count := len(ch.layouts.Layouts) + 1  // Add 1 to allow checking the last possible index
	// Add 1 for the temporary layout [0] if either search result or inverted layout is available
	if ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive) {
		count += 1
	}
	return count
}

// CommandList выводит список всех раскладок
func (ch *CommandHandler) CommandList(args string) error {
	var indicesToPrint []int
	maxIndex := ch.getLayoutCount()

	// Если указаны аргументы, парсим их
	if strings.TrimSpace(args) != "" {
		// Parse ranges but include the search result layout [0] if needed
		parts := strings.Split(strings.TrimSpace(args), ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)

			if strings.Contains(part, "-") {
				// Это диапазон
				rangeParts := strings.Split(part, "-")
				if len(rangeParts) != 2 {
					return fmt.Errorf("неверный формат диапазона: %s", part)
				}

				start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				if err != nil {
					return fmt.Errorf("неверное начало диапазона: %s", rangeParts[0])
				}

				end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err != nil {
					return fmt.Errorf("неверный конец диапазона: %s", rangeParts[1])
				}

				for i := start; i <= end; i++ {
					if i >= 0 && i < maxIndex {
						// Check if this index is valid
						if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
							indicesToPrint = append(indicesToPrint, i)
						} else if i > 0 && i <= len(ch.layouts.Layouts) {
							indicesToPrint = append(indicesToPrint, i)
						}
					}
				}
			} else {
				// Это одиночный индекс
				idx, err := strconv.Atoi(part)
				if err != nil {
					return fmt.Errorf("неверный индекс: %s", part)
				}

				if idx >= 0 && idx < maxIndex {
					// Check if this index is valid
					if idx == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
						indicesToPrint = append(indicesToPrint, idx)
					} else if idx > 0 && idx <= len(ch.layouts.Layouts) {
						indicesToPrint = append(indicesToPrint, idx)
					}
				}
			}
		}
		sort.Ints(indicesToPrint)
	} else {
		// Выводим все раскладки
		for i := 0; i < maxIndex; i++ {
			// Check if this index is valid
			if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
				indicesToPrint = append(indicesToPrint, i)
			} else if i > 0 && i <= len(ch.layouts.Layouts) {
				indicesToPrint = append(indicesToPrint, i)
			}
		}
	}

	for _, idx := range indicesToPrint {
		var layout *Layout
		var found bool

		layout, found = ch.getLayoutByIndex(idx)

		if !found {
			continue
		}

		// Format layout number: for search result layout [0], show it as [0], for others show as [index]
		// Check if layout should be highlighted
		if ch.highlightedLayouts[idx] {
			fmt.Printf("\033[38;2;249;226;175m[%d] %s\033[0m\n", idx, layout.Name)
		} else if idx == 0 {
			fmt.Printf("[%d] %s\n", idx, layout.Name)
		} else {
			fmt.Printf("[%d] %s\n", idx, layout.Name)
		}

		// Находим максимальную частоту для нормирования
		maxFreq := 0.0

		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				key := layout.Keys[row][col]
				if freq, exists := ch.langData.Characters[key]; exists {
					if freq > maxFreq {
						maxFreq = freq
					}
				}
			}
		}

		// Выводим строки раскладки с цветным форматированием
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				if col == 5 {
					fmt.Print(" ")
				}

				key := layout.Keys[row][col]
				freq := 0.0
				if f, exists := ch.langData.Characters[key]; exists {
					freq = f
				}

				// Нормируем частоту на 100%
				// 0% -> цвет (215,215,215) серый
				// 100% -> цвет (215,0,0) красный
				// остальное -> линейная интерполяция

				var r, g, b int
				if maxFreq > 0 {
					percent := (freq / maxFreq) * 100.0
					// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
					// При 0%: (215,215,215), при 100%: (215,0,0)
					r = 215
					g = 215 - int((percent/100.0)*(215.0))
					b = 215 - int((percent/100.0)*(215.0))
				} else {
					// Если нет данных по частоте - серый
					r, g, b = 215, 215, 215
				}

				// Display with frequency-based color
				fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	return nil
}

// CommandLayoutList выводит обе таблицы - со статистикой по нажатиям клавиш и по биграммам
func (ch *CommandHandler) CommandLayoutList(args string) error {
	var indicesToAnalyze []int
	maxIndex := ch.getLayoutCount()

	// Если указаны аргументы, парсим их
	if strings.TrimSpace(args) != "" {
		// Parse ranges but include the search result layout [0] if needed
		parts := strings.Split(strings.TrimSpace(args), ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)

			if strings.Contains(part, "-") {
				// Это диапазон
				rangeParts := strings.Split(part, "-")
				if len(rangeParts) != 2 {
					return fmt.Errorf("неверный формат диапазона: %s", part)
				}

				start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				if err != nil {
					return fmt.Errorf("неверное начало диапазона: %s", rangeParts[0])
				}

				end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err != nil {
					return fmt.Errorf("неверный конец диапазона: %s", rangeParts[1])
				}

				for i := start; i <= end; i++ {
					if i >= 0 && i < maxIndex {
						// Check if this index is valid
						if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
							indicesToAnalyze = append(indicesToAnalyze, i)
						} else if i > 0 && i <= len(ch.layouts.Layouts) {
							indicesToAnalyze = append(indicesToAnalyze, i)
						}
					}
				}
			} else {
				// Это одиночный индекс
				idx, err := strconv.Atoi(part)
				if err != nil {
					return fmt.Errorf("неверный индекс: %s", part)
				}

				if idx >= 0 && idx < maxIndex {
					// Check if this index is valid
					if idx == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
						indicesToAnalyze = append(indicesToAnalyze, idx)
					} else if idx > 0 && idx <= len(ch.layouts.Layouts) {
						indicesToAnalyze = append(indicesToAnalyze, idx)
					}
				}
			}
		}
		sort.Ints(indicesToAnalyze)
	} else {
		// Анализируем все раскладки
		for i := 0; i < maxIndex; i++ {
			// Check if this index is valid
			if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
				indicesToAnalyze = append(indicesToAnalyze, i)
			} else if i > 0 && i <= len(ch.layouts.Layouts) {
				indicesToAnalyze = append(indicesToAnalyze, i)
			}
		}
	}

	// Собираем все анализы
	var analyses []*LayoutAnalysis
	for _, idx := range indicesToAnalyze {
		var layout *Layout
		var found bool

		layout, found = ch.getLayoutByIndex(idx)

		if !found {
			continue
		}

		analysis := AnalyzeLayout(layout, ch.config, ch.langData)
		analysis.LayoutIndex = idx // Set the layout index (0 for search result, 1+ for loaded layouts)
		analyses = append(analyses, analysis)
	}

	// Сортируем по возрастанию значения общей оценки раскладки (WeightedScore)
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].WeightedScore < analyses[j].WeightedScore
	})

	// Найдем индекс лучшей раскладки из загруженных (не [0]), если таковая существует
	bestLoadedLayoutIndex := -1
	bestLoadedScore := math.MaxFloat64
	for _, analysis := range analyses {
		if analysis.LayoutIndex != 0 && analysis.WeightedScore < bestLoadedScore {
			bestLoadedScore = analysis.WeightedScore
			bestLoadedLayoutIndex = analysis.LayoutIndex
		}
	}

	// Выводим заголовок для таблицы статистики по нажатиям клавиш
	fmt.Printf(" %-3s %-16s %5s %5s %5s %5s %5s %5s %5s %5s %6s %5s %5s %6s %5s %5s %4s %5s %7s %7s\n",
		"№", "Layout", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "R1", "R2", "R3", "Left", "Right", "HDI", "FDI", "MEP", "Effort", "Score")
	fmt.Println(strings.Repeat("-", 134))

	// Выводим отсортированные анализы
	for _, analysis := range analyses {
		if analysis.LayoutIndex == 0 {
			// The search result layout [0] gets yellow color (special treatment still applies)
			// If also highlighted by user, yellow takes precedence
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatAnalysis(analysis))
		} else if analysis.LayoutIndex == bestLoadedLayoutIndex {
			// The best loaded layout gets green color (special treatment still applies)
			// If also highlighted by user, green takes precedence
			fmt.Printf("\033[38;2;158;206;88m%s\033[0m\n", FormatAnalysis(analysis))
		} else if ch.highlightedLayouts[analysis.LayoutIndex] {
			// User-highlighted layout gets yellow color
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatAnalysis(analysis))
		} else {
			fmt.Println(FormatAnalysis(analysis))
		}
	}

	// Пустая строка между таблицами
	fmt.Println()

	// Выводим заголовок для таблицы биграмм
	fmt.Printf(" %-3s %-16s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %8s %7s\n",
		"№", "Layout", "SHB", "SFB", "HVB", "FVB", "HDB", "FDB", "HFB", "HSB", "FSB", "LSB", "SRB", "AFI", "AFO", "TIB", "Total", "Score")
	fmt.Println(strings.Repeat("-", 136))

	// Выводим отсортированные анализы в формате таблицы биграмм
	for _, analysis := range analyses {
		if analysis.LayoutIndex == 0 {
			// The search result layout [0] gets yellow color (special treatment still applies)
			// If also highlighted by user, yellow takes precedence
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatBigramAnalysis(analysis))
		} else if analysis.LayoutIndex == bestLoadedLayoutIndex {
			// The best loaded layout gets green color (special treatment still applies)
			// If also highlighted by user, green takes precedence
			fmt.Printf("\033[38;2;158;206;88m%s\033[0m\n", FormatBigramAnalysis(analysis))
		} else if ch.highlightedLayouts[analysis.LayoutIndex] {
			// User-highlighted layout gets yellow color
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatBigramAnalysis(analysis))
		} else {
			fmt.Println(FormatBigramAnalysis(analysis))
		}
	}

	return nil
}

// CommandCoefficients выводит используемые в анализе и поиске коэффициенты из конфигурационного файла с нумерацией
func (ch *CommandHandler) CommandCoefficients(args string) error {
	weights := &ch.config.Weights

	// Выводим только используемые в анализе и поиске коэффициенты с нумерацией
	fmt.Println("Используемые коэффициенты из конфигурационного файла:")
	fmt.Println(" 1. TotalEffortNorm (Нормализующий коэффициент для общего усилия):", weights.TotalEffortNorm)
	fmt.Println(" 2. HDI (Hand Disbalance Index):", weights.HDI)
	fmt.Println(" 3. FDI (Finger Disbalance Index):", weights.FDI)
	fmt.Println(" 4. D18 (Коэффициент дисбаланса между пальцами 1 и 8):", weights.D18)
	fmt.Println(" 5. D27 (Коэффициент дисбаланса между пальцами 2 и 7):", weights.D27)
	fmt.Println(" 6. D36 (Коэффициент дисбаланса между пальцами 3 и 6):", weights.D36)
	fmt.Println(" 7. D45 (Коэффициент дисбаланса между пальцами 4 и 5):", weights.D45)
	fmt.Println(" 8. SHB (Same Hand Bigram):", weights.SHB)
	fmt.Println(" 9. SFB (Same Finger Bigrams):", weights.SFB)
	fmt.Println("10. HVB (Half Vertical Bigrams):", weights.HVB)
	fmt.Println("11. FVB (Full Vertical Bigrams):", weights.FVB)
	fmt.Println("12. HDB (Half Diagonal Bigrams):", weights.HDB)
	fmt.Println("13. FDB (Full Diagonal Bigrams):", weights.FDB)
	fmt.Println("14. HFB (Horizontal Finger Bigrams):", weights.HFB)
	fmt.Println("15. HSB (Half Scissors Bigrams):", weights.HSB)
	fmt.Println("16. FSB (Full Scissors Bigrams):", weights.FSB)
	fmt.Println("17. LSB (Lateral Stretch Bigram):", weights.LSB)
	fmt.Println("18. SRB (Same Row Bigrams):", weights.SRB)
	fmt.Println("19. AFI (Adjacent Fingers In - соседние клавиши в одном ряду нажимаются по направлению к центру):", weights.AFI)
	fmt.Println("20. AFO (Adjacent Fingers Out - соседние клавиши в одном ряду нажимаются по направлению от центра):", weights.AFO)
	fmt.Println("21. HSB_strict_mode (Режим строгой проверки HSB: 1=вкл, 0=выкл):", weights.HSBStrictMode)
	fmt.Println("22. FSB_strict_mode (Режим строгой проверки FSB: 1=вкл, 0=выкл):", weights.FSBStrictMode)
	fmt.Println("23. LSB_strict_mode (Режим строгой проверки LSB: 1=вкл, 0=выкл):", weights.LSBStrictMode)
	fmt.Println("24. MR1 (Максимальное усилие для 1 ряда):", ch.config.Weights.MaxRowEffort1)
	fmt.Println("25. MR2 (Максимальное усилие для 2 ряда):", ch.config.Weights.MaxRowEffort2)
	fmt.Println("26. MR3 (Максимальное усилие для 3 ряда):", ch.config.Weights.MaxRowEffort3)
	fmt.Println("27. PR1 (Штраф для 1 ряда за превышение максимального усилия):", ch.config.Weights.RowPenalty1)
	fmt.Println("28. PR2 (Штраф для 2 ряда за превышение максимального усилия):", ch.config.Weights.RowPenalty2)
	fmt.Println("29. PR3 (Штраф для 3 ряда за превышение максимального усилия):", ch.config.Weights.RowPenalty3)

	// Выводим список индивидуальных коэффициентов биграмм
	if len(ch.config.BigramIndividualCoeffs) > 0 {
		fmt.Println() // Пустая строка перед списком индивидуальных коэффициентов
		fmt.Println("Индивидуальные коэффициенты для отдельных биграмм:")

		// Группируем коэффициенты по значениям
		coeffMap := make(map[float64][]string)
		for _, coeff := range ch.config.BigramIndividualCoeffs {
			key := coeff.Coeff
			bigramStr := fmt.Sprintf("%d-%d", coeff.Pos1+1, coeff.Pos2+1) // +1 для преобразования индексов в номера позиций (1-30)
			coeffMap[key] = append(coeffMap[key], bigramStr)
		}

		// Создаем срез уникальных значений коэффициентов и сортируем его в порядке возрастания
		var uniqueCoeffs []float64
		for coeff := range coeffMap {
			uniqueCoeffs = append(uniqueCoeffs, coeff)
		}
		sort.Float64s(uniqueCoeffs) // Сортируем в порядке возрастания

		// Выводим сгруппированные коэффициенты
		for _, coeff := range uniqueCoeffs {
			bigrams := coeffMap[coeff]
			fmt.Printf("%.1f: %s\n", coeff, strings.Join(bigrams, " "))
		}
	}

	return nil
}

// CommandSetCoefficient устанавливает значение коэффициента по номеру
func (ch *CommandHandler) CommandSetCoefficient(args string) error {
	parts := strings.Fields(args)
	if len(parts) != 2 {
		return fmt.Errorf("используйте: set N значение (где N - номер коэффициента, значение - новое значение)")
	}

	numStr := parts[0]
	valueStr := parts[1]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return fmt.Errorf("некорректный номер коэффициента: %v", err)
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return fmt.Errorf("некорректное значение коэффициента: %v", err)
	}

	weights := &ch.config.Weights

	// Устанавливаем значение коэффициента по номеру (только используемые в анализе и поиске)
	switch num {
	case 1:
		weights.TotalEffortNorm = value
		ch.configTracker.SetWeight("TotalEffortNorm", value)
	case 2:
		weights.HDI = value
		ch.configTracker.SetWeight("HDI", value)
	case 3:
		weights.FDI = value
		ch.configTracker.SetWeight("FDI", value)
	case 4:
		weights.D18 = value
		ch.configTracker.SetWeight("D18", value)
	case 5:
		weights.D27 = value
		ch.configTracker.SetWeight("D27", value)
	case 6:
		weights.D36 = value
		ch.configTracker.SetWeight("D36", value)
	case 7:
		weights.D45 = value
		ch.configTracker.SetWeight("D45", value)
	case 8:
		weights.SHB = value
		ch.configTracker.SetWeight("SHB", value)
	case 9:
		weights.SFB = value
		ch.configTracker.SetWeight("SFB", value)
	case 10:
		weights.HVB = value
		ch.configTracker.SetWeight("HVB", value)
	case 11:
		weights.FVB = value
		ch.configTracker.SetWeight("FVB", value)
	case 12:
		weights.HDB = value
		ch.configTracker.SetWeight("HDB", value)
	case 13:
		weights.FDB = value
		ch.configTracker.SetWeight("FDB", value)
	case 14:
		weights.HFB = value
		ch.configTracker.SetWeight("HFB", value)
	case 15:
		weights.HSB = value
		ch.configTracker.SetWeight("HSB", value)
	case 16:
		weights.FSB = value
		ch.configTracker.SetWeight("FSB", value)
	case 17:
		weights.LSB = value
		ch.configTracker.SetWeight("LSB", value)
	case 18:
		weights.SRB = value
		ch.configTracker.SetWeight("SRB", value)
	case 19:
		weights.AFI = value
		ch.configTracker.SetWeight("AFI", value)
	case 20:
		weights.AFO = value
		ch.configTracker.SetWeight("AFO", value)
	case 21:
		weights.HSBStrictMode = int(value)
		ch.configTracker.SetIntWeight("HSBStrictMode", int(value))
	case 22:
		weights.FSBStrictMode = int(value)
		ch.configTracker.SetIntWeight("FSBStrictMode", int(value))
	case 23:
		weights.LSBStrictMode = int(value)
		ch.configTracker.SetIntWeight("LSBStrictMode", int(value))
	case 24:
		ch.config.Weights.MaxRowEffort1 = value
		ch.config.MaxRowEfforts[0] = value
		ch.configTracker.SetWeight("MaxRowEffort1", value)
		fmt.Printf("MR1 (максимальное усилие для 1 ряда) установлено в значение: %g\n", value)
		return nil
	case 25:
		ch.config.Weights.MaxRowEffort2 = value
		ch.config.MaxRowEfforts[1] = value
		ch.configTracker.SetWeight("MaxRowEffort2", value)
		fmt.Printf("MR2 (максимальное усилие для 2 ряда) установлено в значение: %g\n", value)
		return nil
	case 26:
		ch.config.Weights.MaxRowEffort3 = value
		ch.config.MaxRowEfforts[2] = value
		ch.configTracker.SetWeight("MaxRowEffort3", value)
		fmt.Printf("MR3 (максимальное усилие для 3 ряда) установлено в значение: %g\n", value)
		return nil
	case 27:
		ch.config.Weights.RowPenalty1 = value
		ch.config.RowEffortPenalties[0] = value
		ch.configTracker.SetWeight("RowPenalty1", value)
		fmt.Printf("PR1 (штраф для 1 ряда за превышение максимального усилия) установлено в значение: %g\n", value)
		return nil
	case 28:
		ch.config.Weights.RowPenalty2 = value
		ch.config.RowEffortPenalties[1] = value
		ch.configTracker.SetWeight("RowPenalty2", value)
		fmt.Printf("PR2 (штраф для 2 ряда за превышение максимального усилия) установлено в значение: %g\n", value)
		return nil
	case 29:
		ch.config.Weights.RowPenalty3 = value
		ch.config.RowEffortPenalties[2] = value
		ch.configTracker.SetWeight("RowPenalty3", value)
		fmt.Printf("PR3 (штраф для 3 ряда за превышение максимального усилия) установлено в значение: %g\n", value)
		return nil
	default:
		return fmt.Errorf("номер коэффициента %d вне диапазона (1-28)", num)
	}

	fmt.Printf("Коэффициент %d установлен в значение: %g\n", num, value)
	return nil
}

// CommandReload перезагружает все файлы
func (ch *CommandHandler) CommandReload(langFile, configFile, layoutFile string) error {
	langData, err := LoadLanguageData(langFile)
	if err != nil {
		return err
	}

	config, err := LoadKeyboardConfig(configFile)
	if err != nil {
		return err
	}

	layouts, err := LoadLayouts(layoutFile)
	if err != nil {
		return err
	}

	// Применяем измененные веса к новой конфигурации
	ch.configTracker.ApplyToConfig(config)

	// Если был указан отдельный файл с усилиями, загружаем матрицу усилий из него
	if ch.effortFile != "" && ch.effortFile != configFile {
		effortMatrix, err := LoadEffortMatrix(ch.effortFile)
		if err != nil {
			fmt.Printf("Предупреждение: не удалось загрузить матрицу усилий из файла %s: %v\n", ch.effortFile, err)
		} else {
			// Обновляем матрицу усилий в конфигурации
			config.EffortMatrix = effortMatrix
		}
	}

	ch.langData = langData
	ch.config = config
	ch.layouts = layouts
	ch.analyses = nil
	// Reset the inverted layout active flag since everything is being reloaded
	ch.isInvertedLayoutActive = false

	// Обновляем базовую конфигурацию в существующем трекере, чтобы сохранить информацию об изменениях
	ch.configTracker.UpdateBaseConfig(config.Weights)

	fmt.Println("Файлы перезагружены успешно")
	return nil
}

// CommandInfo анализирует раскладки и выводит информацию
func (ch *CommandHandler) CommandInfo(args string) error {
	var indicesToAnalyze []int
	maxIndex := ch.getLayoutCount()

	// Если указаны аргументы, парсим их
	if strings.TrimSpace(args) != "" {
		// Parse ranges but include the search result layout [0] if needed
		parts := strings.Split(strings.TrimSpace(args), ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)

			if strings.Contains(part, "-") {
				// Это диапазон
				rangeParts := strings.Split(part, "-")
				if len(rangeParts) != 2 {
					return fmt.Errorf("неверный формат диапазона: %s", part)
				}

				start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				if err != nil {
					return fmt.Errorf("неверное начало диапазона: %s", rangeParts[0])
				}

				end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err != nil {
					return fmt.Errorf("неверный конец диапазона: %s", rangeParts[1])
				}

				for i := start; i <= end; i++ {
					if i >= 0 && i < maxIndex {
						// Check if this index is valid
						if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
							indicesToAnalyze = append(indicesToAnalyze, i)
						} else if i > 0 && i <= len(ch.layouts.Layouts) {
							indicesToAnalyze = append(indicesToAnalyze, i)
						}
					}
				}
			} else {
				// Это одиночный индекс
				idx, err := strconv.Atoi(part)
				if err != nil {
					return fmt.Errorf("неверный индекс: %s", part)
				}

				if idx >= 0 && idx < maxIndex {
					// Check if this index is valid
					if idx == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
						indicesToAnalyze = append(indicesToAnalyze, idx)
					} else if idx > 0 && idx <= len(ch.layouts.Layouts) {
						indicesToAnalyze = append(indicesToAnalyze, idx)
					}
				}
			}
		}
		sort.Ints(indicesToAnalyze)
	} else {
		// Анализируем все раскладки
		for i := 0; i < maxIndex; i++ {
			// Check if this index is valid
			if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
				indicesToAnalyze = append(indicesToAnalyze, i)
			} else if i > 0 && i <= len(ch.layouts.Layouts) {
				indicesToAnalyze = append(indicesToAnalyze, i)
			}
		}
	}

	// Собираем все анализы
	var analyses []*LayoutAnalysis
	for _, idx := range indicesToAnalyze {
		var layout *Layout
		var found bool

		layout, found = ch.getLayoutByIndex(idx)

		if !found {
			continue
		}

		analysis := AnalyzeLayout(layout, ch.config, ch.langData)
		analysis.LayoutIndex = idx // Set the layout index (0 for search result, 1+ for loaded layouts)
		analyses = append(analyses, analysis)
	}

	// Сортируем по возрастанию значения общей оценки раскладки (WeightedScore)
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].WeightedScore < analyses[j].WeightedScore
	})

	// Найдем индекс лучшей раскладки из загруженных (не [0]), если таковая существует
	bestLoadedLayoutIndex := -1
	bestLoadedScore := math.MaxFloat64
	for _, analysis := range analyses {
		if analysis.LayoutIndex != 0 && analysis.WeightedScore < bestLoadedScore {
			bestLoadedScore = analysis.WeightedScore
			bestLoadedLayoutIndex = analysis.LayoutIndex
		}
	}

	// Выводим заголовок
	fmt.Printf(" %-3s %-16s %5s %5s %5s %5s %5s %5s %5s %5s %6s %5s %5s %6s %5s %5s %4s %5s %7s %7s\n",
		"№", "Layout", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "R1", "R2", "R3", "Left", "Right", "HDI", "FDI", "MEP", "Effort", "Score")
	fmt.Println(strings.Repeat("-", 134))

	// Выводим отсортированные анализы
	for _, analysis := range analyses {
		if analysis.LayoutIndex == 0 {
			// The search result layout [0] gets yellow color (special treatment still applies)
			// If also highlighted by user, yellow takes precedence
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatAnalysis(analysis))
		} else if analysis.LayoutIndex == bestLoadedLayoutIndex {
			// The best loaded layout gets green color (special treatment still applies)
			// If also highlighted by user, green takes precedence
			fmt.Printf("\033[38;2;158;206;88m%s\033[0m\n", FormatAnalysis(analysis))
		} else if ch.highlightedLayouts[analysis.LayoutIndex] {
			// User-highlighted layout gets yellow color
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatAnalysis(analysis))
		} else {
			fmt.Println(FormatAnalysis(analysis))
		}
	}

	return nil
}

// CommandBigrams анализирует биграммы
func (ch *CommandHandler) CommandBigrams(args string) error {
	var indicesToAnalyze []int
	maxIndex := ch.getLayoutCount()

	// Если указаны аргументы, парсим их
	if strings.TrimSpace(args) != "" {
		// Parse ranges but include the search result layout [0] if needed
		parts := strings.Split(strings.TrimSpace(args), ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)

			if strings.Contains(part, "-") {
				// Это диапазон
				rangeParts := strings.Split(part, "-")
				if len(rangeParts) != 2 {
					return fmt.Errorf("неверный формат диапазона: %s", part)
				}

				start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				if err != nil {
					return fmt.Errorf("неверное начало диапазона: %s", rangeParts[0])
				}

				end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err != nil {
					return fmt.Errorf("неверный конец диапазона: %s", rangeParts[1])
				}

				for i := start; i <= end; i++ {
					if i >= 0 && i < maxIndex {
						// Check if this index is valid
						if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
							indicesToAnalyze = append(indicesToAnalyze, i)
						} else if i > 0 && i <= len(ch.layouts.Layouts) {
							indicesToAnalyze = append(indicesToAnalyze, i)
						}
					}
				}
			} else {
				// Это одиночный индекс
				idx, err := strconv.Atoi(part)
				if err != nil {
					return fmt.Errorf("неверный индекс: %s", part)
				}

				if idx >= 0 && idx < maxIndex {
					// Check if this index is valid
					if idx == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
						indicesToAnalyze = append(indicesToAnalyze, idx)
					} else if idx > 0 && idx <= len(ch.layouts.Layouts) {
						indicesToAnalyze = append(indicesToAnalyze, idx)
					}
				}
			}
		}
		sort.Ints(indicesToAnalyze)
	} else {
		// Анализируем все раскладки
		for i := 0; i < maxIndex; i++ {
			// Check if this index is valid
			if i == 0 && (ch.searchResultLayout != nil || (ch.invertedLayout != nil && ch.isInvertedLayoutActive)) {
				indicesToAnalyze = append(indicesToAnalyze, i)
			} else if i > 0 && i <= len(ch.layouts.Layouts) {
				indicesToAnalyze = append(indicesToAnalyze, i)
			}
		}
	}

	// Собираем все анализы
	var analyses []*LayoutAnalysis
	for _, idx := range indicesToAnalyze {
		var layout *Layout
		var found bool

		layout, found = ch.getLayoutByIndex(idx)

		if !found {
			continue
		}

		analysis := AnalyzeLayout(layout, ch.config, ch.langData)
		analysis.LayoutIndex = idx // Set the layout index (0 for search result, 1+ for loaded layouts)
		analyses = append(analyses, analysis)
	}

	// Сортируем по возрастанию значения общей оценки раскладки (WeightedScore)
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].WeightedScore < analyses[j].WeightedScore
	})

	// Найдем индекс лучшей раскладки из загруженных (не [0]), если таковая существует
	bestLoadedLayoutIndex := -1
	bestLoadedScore := math.MaxFloat64
	for _, analysis := range analyses {
		if analysis.LayoutIndex != 0 && analysis.WeightedScore < bestLoadedScore {
			bestLoadedScore = analysis.WeightedScore
			bestLoadedLayoutIndex = analysis.LayoutIndex
		}
	}

	// Выводим заголовок для таблицы биграмм
	fmt.Printf(" %-3s %-16s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %8s %7s\n",
		"№", "Layout", "SHB", "SFB", "HVB", "FVB", "HDB", "FDB", "HFB", "HSB", "FSB", "LSB", "SRB", "AFI", "AFO", "TIB", "Total", "Score")
	fmt.Println(strings.Repeat("-", 136))

	// Выводим отсортированные анализы в формате таблицы
	for _, analysis := range analyses {
		if analysis.LayoutIndex == 0 {
			// The search result layout [0] gets yellow color (special treatment still applies)
			// If also highlighted by user, green takes precedence
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatBigramAnalysis(analysis))
		} else if analysis.LayoutIndex == bestLoadedLayoutIndex {
			// The best loaded layout gets green color (special treatment still applies)
			// If also highlighted by user, green takes precedence
			fmt.Printf("\033[38;2;158;206;88m%s\033[0m\n", FormatBigramAnalysis(analysis))
		} else if ch.highlightedLayouts[analysis.LayoutIndex] {
			// User-highlighted layout gets yellow color
			fmt.Printf("\033[38;2;249;226;175m%s\033[0m\n", FormatBigramAnalysis(analysis))
		} else {
			fmt.Println(FormatBigramAnalysis(analysis))
		}
	}

	return nil
}

// CommandHighlight управляет подсветкой раскладок
func (ch *CommandHandler) CommandHighlight(args string) error {
	if strings.TrimSpace(args) == "" {
		// Если аргументов нет, снимаем все подсветки
		for k := range ch.highlightedLayouts {
			delete(ch.highlightedLayouts, k)
		}
		fmt.Println("Все подсветки раскладок сняты")
		return nil
	}

	var indicesToHighlight []int

	// Парсим аргументы - это может быть список через запятую и/или диапазоны
	parts := strings.Split(strings.TrimSpace(args), ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if strings.Contains(part, "-") {
			// Это диапазон
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return fmt.Errorf("неверный формат диапазона: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return fmt.Errorf("неверное начало диапазона: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return fmt.Errorf("неверный конец диапазона: %s", rangeParts[1])
			}

			if start > end {
				start, end = end, start // Поменять местами, если начальный индекс больше конечного
			}

			for i := start; i <= end; i++ {
				totalLayouts := ch.getLayoutCount()
				if i >= 0 && i < totalLayouts { // Индексация с 0
					indicesToHighlight = append(indicesToHighlight, i)
				} else {
					fmt.Printf("Предупреждение: номер раскладки %d вне диапазона (допустимо от 0 до %d)\n", i, totalLayouts-1)
				}
			}
		} else {
			// Это одиночный индекс
			idx, err := strconv.Atoi(part)
			if err != nil {
				return fmt.Errorf("неверный индекс: %s", part)
			}

			totalLayouts := ch.getLayoutCount()
			if idx >= 0 && idx < totalLayouts { // Индексация с 0
				indicesToHighlight = append(indicesToHighlight, idx)
			} else {
				return fmt.Errorf("номер раскладки %d вне диапазона (допустимо от 0 до %d)", idx, totalLayouts-1)
			}
		}
	}

	// Переключаем подсветку для всех указанных раскладок
	for _, idx := range indicesToHighlight {
		// Переключаем подсветку
		if ch.highlightedLayouts[idx] {
			delete(ch.highlightedLayouts, idx)
			fmt.Printf("Подсветка с раскладки [%d] снята\n", idx)
		} else {
			ch.highlightedLayouts[idx] = true
			fmt.Printf("Раскладка [%d] подсвечена\n", idx)
		}
	}

	return nil
}

// ParseCommand парсит и выполняет команду
func (ch *CommandHandler) ParseCommand(cmd string) error {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	// Парсим команду и аргументы
	parts := strings.SplitN(cmd, " ", 2)
	command := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch command {
	case "p":
		return ch.CommandList(args)
	case "l":
		return ch.CommandInfo(args)
	case "r":
		return ch.CommandReload(ch.langFile, ch.configFile, ch.layoutFile)
	case "lb":
		return ch.CommandBigrams(args)
	case "ll":
		return ch.CommandLayoutList(args)
	case "c":
		return ch.CommandCoefficients(args)
	case "set":
		return ch.CommandSetCoefficient(args)
	case "s":
		return ch.CommandSave(args)
	case "sort":
		return ch.CommandSort(args)
	case "g":
		return ch.CommandAnalyze(args)
	case "gg":
		return ch.CommandContinuousAnalyze(args)
	case "inv":
		return ch.CommandInvert(args)
	case "sw":
		return ch.CommandSwapLetters(args)
	case "d":
		return ch.CommandDelete(args)
	case "n":
		return ch.CommandRename(args)
	case "h":
		return ch.CommandHighlight(args)
	case "a":
		return ch.CommandLayoutAnalysis(args)
	case "t":
		return ch.CommandDetailedInfo(args)
	case "b":
		return ch.CommandBigramLetter(args)
	case "help":
		printHelp()
	case "exit", "quit", "q":
		return nil
	default:
		return fmt.Errorf("неизвестная команда: %s", command)
	}

	return nil
}

// CommandAnalyze выполняет поиск оптимальной раскладки
func (ch *CommandHandler) CommandAnalyze(args string) error {
	// Сброс всех временных раскладок перед началом нового поиска
	ch.searchResultLayout = nil
	ch.invertedLayout = nil
	ch.isInvertedLayoutActive = false
	ch.bestResults = make([]SimulatedAnnealingResult, 0)

	// Parse arguments
	var layoutNumber int = 0  // Layout number to start from (0 for random search)
	var shouldUseRandomLayout bool = false  // Flag for random search
	numBest := 1

	if strings.TrimSpace(args) != "" {
		parts := strings.Fields(strings.TrimSpace(args))
		if len(parts) == 1 {
			num, err := strconv.Atoi(parts[0])
			if err == nil && num > 0 {
				// If number is valid and within range of available layouts, treat as layout number
				if num <= len(ch.layouts.Layouts) {
					layoutNumber = num
					numBest = 1 // Default to 1 result when only layout number is specified
				} else {
					// If number exceeds number of layouts, treat as number of results for random search
					layoutNumber = 0 // Indicates random search
					numBest = num
					shouldUseRandomLayout = true
				}
			} else {
				// Invalid number provided
				return fmt.Errorf("некорректный номер раскладки: %s", parts[0])
			}
		} else if len(parts) == 2 {
			// Both layout number and number of best specified
			layoutNum, err1 := strconv.Atoi(parts[0])
			num, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && layoutNum > 0 && num > 0 {
				// Check if layout number is valid
				if layoutNum <= len(ch.layouts.Layouts) {
					layoutNumber = layoutNum
					numBest = num
				} else {
					return fmt.Errorf("номер раскладки %d вне диапазона", layoutNum)
				}
			} else {
				return fmt.Errorf("некорректный формат параметров")
			}
		} else {
			return fmt.Errorf("некорректное количество параметров")
		}
	} else {
		// No arguments - random search
		shouldUseRandomLayout = true
		layoutNumber = 0 // Indicates random search
		numBest = 1
	}

	// Determine the appropriate output message based on whether we're using a random layout or a specific one
	var results []SimulatedAnnealingResult
	params := DefaultSAParams()

	if shouldUseRandomLayout {
		fmt.Printf("Поиск оптимальной раскладки (Simulated Annealing) - исходная раскладка [случайная], выведет %d лучших результатов\n", numBest)
		// Use random layout search instead of existing layout, using only characters from existing layouts
		results = SearchOptimalLayoutFromRandomLayout(ch.config, ch.langData, ch.layouts, params, numBest)
	} else {
		// Specific layout was provided
		var startLayout Layout
		if layoutNumber > 0 && layoutNumber <= len(ch.layouts.Layouts) {
			// Use the specified layout as the starting layout
			startLayout = ch.layouts.Layouts[layoutNumber-1] // Convert to 0-based index
		} else {
			// Fallback: use the layout with lowest weighted score if not found
			bestIndex := 0
			bestScore := math.MaxFloat64
			for i, layout := range ch.layouts.Layouts {
				analysis := AnalyzeLayout(&layout, ch.config, ch.langData)
				if analysis.WeightedScore < bestScore {
					bestScore = analysis.WeightedScore
					bestIndex = i
				}
			}
			startLayout = ch.layouts.Layouts[bestIndex]
		}
		fmt.Printf("Поиск оптимальной раскладки (Simulated Annealing) - исходная раскладка [%d], выведет %d лучших результатов\n", layoutNumber, numBest)
		// Use the starting layout for the search
		results = SearchOptimalLayoutFromSpecificLayout(ch.config, ch.langData, ch.layouts, params, numBest, startLayout)
	}

	// Check if any of the found layouts match existing layouts
	newLayoutFound := false
	for _, result := range results {
		isExisting := false
		for _, existingLayout := range ch.layouts.Layouts {
			if result.Layout.Equals(&existingLayout) {
				isExisting = true
				break
			}
		}
		if !isExisting {
			newLayoutFound = true
			break
		}
	}

	// Store the results for potential inversion regardless of whether they are new
	ch.bestResults = results

	// Store the best result as the search result layout to allow saving it
	if len(results) > 0 {
		// Store the first (best) result
		bestLayout := results[0].Layout
		// Update the name to include [0] to indicate it's the temporary result
		if !strings.Contains(bestLayout.Name, "[0]") && !strings.Contains(bestLayout.Name, "search result") {
			bestLayout.Name = "[0] " + bestLayout.Name
		}
		ch.searchResultLayout = &bestLayout
	}

	if !newLayoutFound {
		fmt.Println("Новая раскладка не найдена")
		return nil
	}

	fmt.Printf("\n%-4s %-16s %7s\n", "№", "Layout", "Score")
	fmt.Println(strings.Repeat("-", 29))

	for i, result := range results {
		fmt.Printf("%-4s %-16s %7.2f\n", fmt.Sprintf("[%d]", i+1), result.Layout.Name, result.Score)

		// Выводим раскладку с цветом
		// Находим максимальную частоту для нормирования
		maxFreq := 0.0

		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				key := result.Layout.Keys[row][col]
				if freq, exists := ch.langData.Characters[key]; exists {
					if freq > maxFreq {
						maxFreq = freq
					}
				}
			}
		}

		// Выводим строки раскладки с цветным форматированием
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				if col == 5 {
					fmt.Print(" ")
				}

				key := result.Layout.Keys[row][col]
				freq := 0.0
				if f, exists := ch.langData.Characters[key]; exists {
					freq = f
				}

				// Нормируем частоту на 100%
				// 0% -> цвет (215,215,215) серый
				// 100% -> цвет (215,0,0) красный
				// остальное -> линейная интерполяция

				var r, g, b int
				if maxFreq > 0 {
					percent := (freq / maxFreq) * 100.0
					// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
					// При 0%: (215,215,215), при 100%: (215,0,0)
					r = 215
					g = 215 - int((percent/100.0)*(215.0))
					b = 215 - int((percent/100.0)*(215.0))
				} else {
					// Если нет данных по частоте - серый
					r, g, b = 215, 215, 215
				}

				fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
			}
			fmt.Println()
		}

		// Выводим анализ
		fmt.Println(FormatAnalysis(result.Analysis))
		fmt.Println()
	}

	return nil
}

// CommandContinuousAnalyze выполняет непрерывный поиск оптимальных раскладок
func (ch *CommandHandler) CommandContinuousAnalyze(args string) error {
	// Сброс всех временных раскладок перед началом нового поиска
	ch.searchResultLayout = nil
	ch.invertedLayout = nil
	ch.isInvertedLayoutActive = false

	// Parse arguments
	var layoutNumber int = 0  // Layout number to start from (0 for random search)
	var shouldUseRandomLayout bool = false  // Flag for random search
	var fileName string = ""  // Optional file name to save results
	numBest := 1

	if strings.TrimSpace(args) != "" {
		parts := strings.Fields(strings.TrimSpace(args))
		if len(parts) == 1 {
			num, err := strconv.Atoi(parts[0])
			if err == nil && num > 0 {
				// If number is valid and within range of available layouts, treat as layout number
				if num <= len(ch.layouts.Layouts) {
					layoutNumber = num
					numBest = 1 // Default to 1 result when only layout number is specified
				} else {
					// If number exceeds number of layouts, treat as number of results for random search
					layoutNumber = 0 // Indicates random search
					numBest = num
					shouldUseRandomLayout = true
				}
			} else {
				// If it's not a number, treat as file name for random search
				fileName = parts[0]
				shouldUseRandomLayout = true
				layoutNumber = 0 // Indicates random search
				numBest = 1
			}
		} else if len(parts) == 2 {
			// Could be layout number + number of best, or layout number + filename, or number of best + filename
			num1, err1 := strconv.Atoi(parts[0])
			num2, err2 := strconv.Atoi(parts[1])

			if err1 == nil && err2 == nil && num1 > 0 && num2 > 0 {
				// Both are numbers: layout number + number of best
				layoutNum := num1
				num := num2
				// Check if layout number is valid
				if layoutNum <= len(ch.layouts.Layouts) {
					layoutNumber = layoutNum
					numBest = num
				} else {
					return fmt.Errorf("номер раскладки %d вне диапазона", layoutNum)
				}
			} else if err1 == nil && num1 > 0 && num2 <= 0 {
				// First is a number (layout number), second is not a valid positive number
				// So second should be treated as filename
				layoutNum := num1
				if layoutNum <= len(ch.layouts.Layouts) {
					layoutNumber = layoutNum
					fileName = parts[1]
					numBest = 1
				} else {
					return fmt.Errorf("номер раскладки %d вне диапазона", layoutNum)
				}
			} else if err1 == nil && num1 > len(ch.layouts.Layouts) && err2 != nil {
				// First number exceeds layout count and second is not a number
				// So first is number of results for random search, second is filename
				layoutNumber = 0 // Indicates random search
				numBest = num1
				shouldUseRandomLayout = true
				fileName = parts[1]
			} else {
				// First is not a number, so it's filename and second is number of results
				fileName = parts[0]
				num, err := strconv.Atoi(parts[1])
				if err == nil && num > 0 {
					layoutNumber = 0 // Indicates random search
					numBest = num
					shouldUseRandomLayout = true
				} else {
					return fmt.Errorf("некорректный формат параметров")
				}
			}
		} else if len(parts) == 3 {
			// layout number + number of best + filename
			layoutNum, err1 := strconv.Atoi(parts[0])
			num, err2 := strconv.Atoi(parts[1])
			if err1 == nil && err2 == nil && layoutNum > 0 && num > 0 {
				// Check if layout number is valid
				if layoutNum <= len(ch.layouts.Layouts) {
					layoutNumber = layoutNum
					numBest = num
					fileName = parts[2]
				} else {
					return fmt.Errorf("номер раскладки %d вне диапазона", layoutNum)
				}
			} else {
				return fmt.Errorf("некорректный формат параметров")
			}
		} else {
			return fmt.Errorf("некорректное количество параметров")
		}
	} else {
		// No arguments - random search
		shouldUseRandomLayout = true
		layoutNumber = 0 // Indicates random search
		numBest = 1
	}

	// Determine the appropriate output message based on whether we're using a random layout or a specific one
	params := DefaultSAParams()

	// Initialize best results
	bestResults := make([]SimulatedAnnealingResult, 0, numBest)

	fmt.Printf("Непрерывный поиск оптимальной раскладки (Simulated Annealing) - начат\n")
	fmt.Println("\x1b[38;2;215;100;100m\nДля остановки поиска нажмите клавишу Q или Esc.\n\x1b[0m")
	fmt.Printf("Параметры: исходная раскладка %s, количество результатов: %d\n",
		func() string {
			if shouldUseRandomLayout {
				return "[случайная]"
			} else {
				return fmt.Sprintf("[%d]", layoutNumber)
			}
		}(), numBest)

	// Открываем клавиатуру в неблокирующем режиме
	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()

	// Канал для сигнала завершения
	done := make(chan struct{})

	// Горутина: ждём нажатие 'q' или Esc
	go func() {
		for {
			char, key, err := keyboard.GetKey()
			if err != nil {
				panic(err)
			}

			if key == 3 || char == 3 || char == 'q' || char == 'Q' || key == keyboard.KeyEsc {
				fmt.Println("\x1b[38;2;215;100;100m\nПолучен сигнал остановки поиска. Дождитесь завершения итерации...\n\x1b[0m")
				close(done)
				return
			}
		}
	}()

	iteration := 0
	stopRequested := false

	for !stopRequested {
		iteration++
		fmt.Printf("\n--- Итерация %d ---\n", iteration)

		var results []SimulatedAnnealingResult

		if shouldUseRandomLayout {
			results = SearchOptimalLayoutFromRandomLayout(ch.config, ch.langData, ch.layouts, params, numBest)
		} else {
			var startLayout Layout
			if layoutNumber > 0 && layoutNumber <= len(ch.layouts.Layouts) {
				// Use the specified layout as the starting layout
				startLayout = ch.layouts.Layouts[layoutNumber-1] // Convert to 0-based index
			} else {
				// Fallback: use the layout with lowest weighted score if not found
				bestIndex := 0
				bestScore := math.MaxFloat64
				for i, layout := range ch.layouts.Layouts {
					analysis := AnalyzeLayout(&layout, ch.config, ch.langData)
					if analysis.WeightedScore < bestScore {
						bestScore = analysis.WeightedScore
						bestIndex = i
					}
				}
				startLayout = ch.layouts.Layouts[bestIndex]
			}
			// Use the starting layout for the search
			results = SearchOptimalLayoutFromSpecificLayout(ch.config, ch.langData, ch.layouts, params, numBest, startLayout)
		}

		select {
		case <-done:
			stopRequested = true
			continue
		default:
			// Продолжаем цикл
		}

		// Check if any of the found layouts are new (not in existing layouts) and better than current best
		newBetterLayoutFound := false
		for _, result := range results {
			isExisting := false
			for _, existingLayout := range ch.layouts.Layouts {
				if result.Layout.Equals(&existingLayout) {
					isExisting = true
					break
				}
			}

			// Check if this result is better than current best results
			isBetterThanBest := false
			if len(bestResults) == 0 {
				isBetterThanBest = true
			} else {
				// Compare with the best result in current bestResults
				isBetterThanBest = result.Score < bestResults[0].Score
			}

			if !isExisting && isBetterThanBest {
				newBetterLayoutFound = true
				break
			}
		}

		if newBetterLayoutFound {
			fmt.Println("Найдена новая раскладка!")
			bestResults = results
		} else {
			fmt.Println("Новая раскладка не найдена или не лучше текущих")
		}

		// Display best result so far
		if len(bestResults) > 0 {
			fmt.Printf("\nЛучшая раскладка на данный момент (итерация %d):\n", iteration)
			bestResult := bestResults[0]
			fmt.Printf("%-4s %-16s %7s\n", "№", "Layout", "Score")
			fmt.Println(strings.Repeat("-", 29))
			fmt.Printf("%-4s %-16s %7.2f\n", "[0]", bestResult.Layout.Name, bestResult.Score)

			// Выводим раскладку с цветом
			// Находим максимальную частоту для нормирования
			maxFreq := 0.0
			for row := 0; row < 3; row++ {
				for col := 0; col < 10; col++ {
					key := bestResult.Layout.Keys[row][col]
					if freq, exists := ch.langData.Characters[key]; exists {
						if freq > maxFreq {
							maxFreq = freq
						}
					}
				}
			}

			// Выводим строки раскладки с цветным форматированием
			for row := 0; row < 3; row++ {
				for col := 0; col < 10; col++ {
					if col == 5 {
						fmt.Print(" ")
					}

					key := bestResult.Layout.Keys[row][col]
					freq := 0.0
					if f, exists := ch.langData.Characters[key]; exists {
						freq = f
					}

					// Нормируем частоту на 100%
					// 0% -> цвет (215,215,215) серый
					// 100% -> цвет (215,0,0) красный
					// остальное -> линейная интерполяция

					var r, g, b int
					if maxFreq > 0 {
						percent := (freq / maxFreq) * 100.0
						// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
						// При 0%: (215,215,215), при 100%: (215,0,0)
						r = 215
						g = 215 - int((percent/100.0)*(215.0))
						b = 215 - int((percent/100.0)*(215.0))
					} else {
						// Если нет данных по частоте - серый
						r, g, b = 215, 215, 215
					}

					fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
				}
				fmt.Println()
			}

			// Выводим анализ
			fmt.Println(FormatAnalysis(bestResult.Analysis))

			// Если указано имя файла и найдена новая лучшая раскладка, записываем её в файл
			if fileName != "" && newBetterLayoutFound {
				// Записываем раскладку в файл
				if err := ch.saveLayoutToFile(bestResult.Layout, fileName); err != nil {
					fmt.Printf("Ошибка записи раскладки в файл %s: %v\n", fileName, err)
				}
			}
		}
	}

	// Store the final best results
	ch.bestResults = bestResults

	// Store the best result as the search result layout to allow saving it
	if len(bestResults) > 0 {
		// Store the first (best) result
		bestLayout := bestResults[0].Layout
		// Update the name to include [0] to indicate it's the temporary result
		if !strings.Contains(bestLayout.Name, "[0]") && !strings.Contains(bestLayout.Name, "search result") {
			bestLayout.Name = "[0] " + bestLayout.Name
		}
		ch.searchResultLayout = &bestLayout

		// Если указано имя файла, записываем финальный результат в файл
		if fileName != "" {
			// Записываем раскладку в файл
			if err := ch.saveLayoutToFile(bestLayout, fileName); err != nil {
				fmt.Printf("Ошибка записи финальной раскладки в файл %s: %v\n", fileName, err)
			}
		}
	}

	fmt.Printf("\nНепрерывный поиск завершен после %d итераций.\n", iteration)
	if len(bestResults) > 0 {
		fmt.Println("Окончательный результат:")
		fmt.Printf("%-4s %-16s %7s\n", "№", "Layout", "Score")
		fmt.Println(strings.Repeat("-", 29))
		for i, result := range bestResults {
			fmt.Printf("%-4s %-16s %7.2f\n", fmt.Sprintf("[%d]", i+1), result.Layout.Name, result.Score)

			// Выводим раскладку с цветом
			// Находим максимальную частоту для нормирования
			maxFreq := 0.0
			for row := 0; row < 3; row++ {
				for col := 0; col < 10; col++ {
					key := result.Layout.Keys[row][col]
					if freq, exists := ch.langData.Characters[key]; exists {
						if freq > maxFreq {
							maxFreq = freq
						}
					}
				}
			}

			// Выводим строки раскладки с цветным форматированием
			for row := 0; row < 3; row++ {
				for col := 0; col < 10; col++ {
					if col == 5 {
						fmt.Print(" ")
					}

					key := result.Layout.Keys[row][col]
					freq := 0.0
					if f, exists := ch.langData.Characters[key]; exists {
						freq = f
					}

					// Нормируем частоту на 100%
					// 0% -> цвет (215,215,215) серый
					// 100% -> цвет (215,0,0) красный
					// остальное -> линейная интерполяция

					var r, g, b int
					if maxFreq > 0 {
						percent := (freq / maxFreq) * 100.0
						// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
						// При 0%: (215,215,215), при 100%: (215,0,0)
						r = 215
						g = 215 - int((percent/100.0)*(215.0))
						b = 215 - int((percent/100.0)*(215.0))
					} else {
						// Если нет данных по частоте - серый
						r, g, b = 215, 215, 215
					}

					fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
				}
				fmt.Println()
			}

			// Выводим анализ
			fmt.Println(FormatAnalysis(result.Analysis))
			fmt.Println()
		}
	} else {
		fmt.Println("Не найдено ни одной раскладки.")
	}

	return nil
}

// CommandInvert выводит указанную раскладку в инвертированном виде (зеркально относительно центра)
func (ch *CommandHandler) CommandInvert(args string) error {
	// Если аргументы не указаны, проверяем есть ли результаты поиска
	if strings.TrimSpace(args) == "" {
		// Проверяем наличие результатов поиска
		if len(ch.bestResults) == 0 {
			fmt.Println("Не указана раскладка для инвертирования")
			return nil
		}

		// Используем первую (лучшую) раскладку из результатов поиска
		layout := ch.bestResults[0].Layout

		// Создаем инвертированную (зеркальную) копию раскладки
		invertedLayout := Layout{
			Name: layout.Name + " (inv)",
			Keys: [3][10]string{},
		}

		// Зеркально отражаем раскладку по вертикали (относительно центра между раскладками)
		// Это означает отражение по горизонтали: колонка i становится колонкой (9-i)
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				invertedLayout.Keys[row][col] = layout.Keys[row][9-col]
			}
		}

		fmt.Printf("\n%s\n", invertedLayout.Name)
		fmt.Println(strings.Repeat("-", len(invertedLayout.Name)))

		// Находим максимальную частоту для нормирования цвета
		maxFreq := 0.0
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				key := invertedLayout.Keys[row][col]
				if freq, exists := ch.langData.Characters[key]; exists {
					if freq > maxFreq {
						maxFreq = freq
					}
				}
			}
		}

		// Store the inverted layout for potential access
		ch.invertedLayout = &invertedLayout
		// Set the flag to indicate inverted layout is active
		ch.isInvertedLayoutActive = true

		// Note: We don't update searchResultLayout here to avoid conflict with [0] temporary layout
		// If user wants to save the inverted layout, they should use 's' immediately after 'inv'

		// Выводим строки раскладки с цветным форматированием
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				if col == 5 {
					fmt.Print(" ")
				}

				key := invertedLayout.Keys[row][col]
				freq := 0.0
				if f, exists := ch.langData.Characters[key]; exists {
					freq = f
				}

				// Нормируем частоту на 100%
				// 0% -> цвет (215,215,215) серый
				// 100% -> цвет (215,0,0) красный
				// остальное -> линейная интерполяция

				var r, g, b int
				if maxFreq > 0 {
					percent := (freq / maxFreq) * 100.0
					// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
					// При 0%: (215,215,215), при 100%: (215,0,0)
					r = 215
					g = 215 - int((percent/100.0)*(215.0))
					b = 215 - int((percent/100.0)*(215.0))
				} else {
					// Если нет данных по частоте - серый
					r, g, b = 215, 215, 215
				}

				fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
			}
			fmt.Println()
		}
		fmt.Println()
		return nil
	}

	// Иначе работаем со старой логикой - по индексам
	indices := []int{}
	parts := strings.Split(args, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
				if err1 == nil && err2 == nil && start <= end && start > 0 {
					for i := start - 1; i < end && i < len(ch.layouts.Layouts); i++ {
						indices = append(indices, i)
					}
				}
			}
		} else {
			num, err := strconv.Atoi(part)
			if err == nil {
				if num == 0 {
					// Обработка специального случая: инвертирование активной раскладки под индексом 0
					var layoutToInvert *Layout
					if ch.isInvertedLayoutActive && ch.invertedLayout != nil {
						// Если активна инвертированная раскладка, инвертируем её (двойная инверсия = оригинальная)
						layoutToInvert = ch.invertedLayout
					} else if ch.searchResultLayout != nil {
						// Если есть результат поиска, инвертируем его
						layoutToInvert = ch.searchResultLayout
					} else {
						// Нет активной раскладки под индексом 0
						fmt.Printf("Нет раскладки с индексом 0 для инвертирования\n")
						return nil
					}

					// Создаем инвертированную (зеркальную) копию раскладки
					invertedLayout := Layout{
						Name: layoutToInvert.Name + " (inverted again)",
						Keys: [3][10]string{},
					}

					// Зеркально отражаем раскладку по вертикали (относительно центра между раскладками)
					// Это означает отражение по горизонтали: колонка i становится колонкой (9-i)
					for row := 0; row < 3; row++ {
						for col := 0; col < 10; col++ {
							invertedLayout.Keys[row][col] = layoutToInvert.Keys[row][9-col]
						}
					}

					fmt.Printf("\n%s\n", invertedLayout.Name)
					fmt.Println(strings.Repeat("-", len(invertedLayout.Name)))

					// Находим максимальную частоту для нормирования цвета
					maxFreq := 0.0
					for row := 0; row < 3; row++ {
						for col := 0; col < 10; col++ {
							key := invertedLayout.Keys[row][col]
							if freq, exists := ch.langData.Characters[key]; exists {
								if freq > maxFreq {
									maxFreq = freq
								}
							}
						}
					}

					// Store the inverted layout for potential access
					ch.invertedLayout = &invertedLayout
					// Set the flag to indicate inverted layout is active
					ch.isInvertedLayoutActive = true

					// Выводим строки раскладки с цветным форматированием
					for row := 0; row < 3; row++ {
						for col := 0; col < 10; col++ {
							if col == 5 {
								fmt.Print(" ")
							}

							key := invertedLayout.Keys[row][col]
							freq := 0.0
							if f, exists := ch.langData.Characters[key]; exists {
								freq = f
							}

							// Нормируем частоту на 100%
							// 0% -> цвет (215,215,215) серый
							// 100% -> цвет (215,0,0) красный
							// остальное -> линейная интерполяция

							var r, g, b int
							if maxFreq > 0 {
								percent := (freq / maxFreq) * 100.0
								// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
								// При 0%: (215,215,215), при 100%: (215,0,0)
								r = 215
								g = 215 - int((percent/100.0)*(215.0))
								b = 215 - int((percent/100.0)*(215.0))
							} else {
								// Если нет данных по частоте - серый
								r, g, b = 215, 215, 215
							}

							fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
						}
						fmt.Println()
					}
					fmt.Println()
					return nil
				} else if num > 0 {
					if num <= len(ch.layouts.Layouts) {
						indices = append(indices, num-1) // Convert to 0-based index
					}
				}
			}
		}
	}

	// Выводим инвертированные раскладки
	for _, idx := range indices {
		if idx < 0 || idx >= len(ch.layouts.Layouts) {
			continue
		}

		layout := ch.layouts.Layouts[idx]

		// Создаем инвертированную (зеркальную) копию раскладки
		invertedLayout := Layout{
			Name: layout.Name + " (inv)",
			Keys: [3][10]string{},
		}

		// Зеркально отражаем раскладку по вертикали (относительно центра между раскладками)
		// Это означает отражение по горизонтали: колонка i становится колонкой (9-i)
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				invertedLayout.Keys[row][col] = layout.Keys[row][9-col]
			}
		}

		fmt.Printf("\n%s\n", invertedLayout.Name)
		fmt.Println(strings.Repeat("-", len(invertedLayout.Name)))

		// Находим максимальную частоту для нормирования цвета
		maxFreq := 0.0
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				key := invertedLayout.Keys[row][col]
				if freq, exists := ch.langData.Characters[key]; exists {
					if freq > maxFreq {
						maxFreq = freq
					}
				}
			}
		}

		// Store the inverted layout for potential save operation (use the last one if multiple layouts are being inverted)
		ch.invertedLayout = &invertedLayout
		// Set the flag to indicate inverted layout is active
		ch.isInvertedLayoutActive = true
		// Note: We don't update searchResultLayout here to avoid conflict with [0] temporary layout
		// If user wants to save the inverted layout, they should use 's' immediately after 'inv'

		// Выводим строки раскладки с цветным форматированием
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				if col == 5 {
					fmt.Print(" ")
				}

				key := invertedLayout.Keys[row][col]
				freq := 0.0
				if f, exists := ch.langData.Characters[key]; exists {
					freq = f
				}

				// Нормируем частоту на 100%
				// 0% -> цвет (215,215,215) серый
				// 100% -> цвет (215,0,0) красный
				// остальное -> линейная интерполяция

				var r, g, b int
				if maxFreq > 0 {
					percent := (freq / maxFreq) * 100.0
					// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
					// При 0%: (215,215,215), при 100%: (215,0,0)
					r = 215
					g = 215 - int((percent/100.0)*(215.0))
					b = 215 - int((percent/100.0)*(215.0))
				} else {
					// Если нет данных по частоте - серый
					r, g, b = 215, 215, 215
				}

				fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	return nil
}

// CommandSave сохраняет указанную раскладку в конец файла раскладок
func (ch *CommandHandler) CommandSave(args string) error {
	var layoutToSave Layout

	arg := strings.TrimSpace(args)

	if arg == "" {
		// If no argument provided, save the active layout at index [0]
		activeLayout, exists := ch.getLayoutByIndex(0)
		if !exists || activeLayout == nil {
			return fmt.Errorf("не указана раскладка для сохранения")
		}
		layoutToSave = *activeLayout
	} else {
		// If it's a numeric argument (layout number), save that specific layout
		num, err := strconv.Atoi(arg)
		if err == nil && num >= 0 && num <= len(ch.layouts.Layouts) {
			if num == 0 {
				// Save the active layout at index [0]
				activeLayout, exists := ch.getLayoutByIndex(0)
				if !exists || activeLayout == nil {
					return fmt.Errorf("номер раскладки %d не существует", num)
				}
				layoutToSave = *activeLayout
				// Remove the [0] prefix from the name if present
				if strings.HasPrefix(layoutToSave.Name, "[0] ") {
					layoutToSave.Name = layoutToSave.Name[4:] // Remove [0] prefix
				}
			} else if num > 0 && num <= len(ch.layouts.Layouts) {
				// It's a valid layout number - save that specific layout
				layoutToSave = ch.layouts.Layouts[num-1]  // Convert to 0-based index
			} else {
				return fmt.Errorf("номер раскладки %d вне диапазона", num)
			}
		} else {
			// It's a name parameter - save the active layout at index [0] with this name
			activeLayout, exists := ch.getLayoutByIndex(0)
			if !exists || activeLayout == nil {
				return fmt.Errorf("не указана раскладка для сохранения")
			}
			layoutToSave = *activeLayout
			layoutToSave.Name = arg
		}
	}

	// Открываем файл для добавления
	file, err := os.OpenFile(ch.outputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла %s для добавления: %v", ch.outputFile, err)
	}
	defer file.Close()

	// Записываем пустую строку-разделитель перед новой раскладкой
	// Проверяем, чтобы новые раскладки отделялись от уже имеющихся в файле пустой строкой
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("ошибка записи разделителя: %v", err)
	}

	// Записываем имя раскладки
	if _, err := file.WriteString(fmt.Sprintf("%s\n", layoutToSave.Name)); err != nil {
		return fmt.Errorf("ошибка записи имени раскладки: %v", err)
	}

	// Записываем строки раскладки
	for row := 0; row < 3; row++ {
		line := ""
		for col := 0; col < 10; col++ {
			if col > 0 {
				line += " "
			}
			line += layoutToSave.Keys[row][col]
			// Добавляем дополнительный пробел между половинками (между 5 и 6 столбцом)
			if col == 4 {
				line += " "
			}
		}
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("ошибка записи строки раскладки: %v", err)
		}
	}

	// Перезагружаем файл раскладок, чтобы обновить внутреннее состояние
	newLangData, newConfig, newParsedLayouts, err := LoadAllData(ch.langFile, ch.configFile, ch.outputFile)
	if err != nil {
		return fmt.Errorf("ошибка перезагрузки данных: %v", err)
	}

	// Применяем измененные веса к новой конфигурации
	ch.configTracker.ApplyToConfig(newConfig)

	// Если был указан отдельный файл с усилиями, загружаем матрицу усилий из него
	if ch.effortFile != "" && ch.effortFile != ch.configFile {
		effortMatrix, err := LoadEffortMatrix(ch.effortFile)
		if err != nil {
			fmt.Printf("Предупреждение: не удалось загрузить матрицу усилий из файла %s: %v\n", ch.effortFile, err)
		} else {
			// Обновляем матрицу усилий в конфигурации
			newConfig.EffortMatrix = effortMatrix
		}
	}

	ch.langData = newLangData
	ch.config = newConfig
	ch.layouts = newParsedLayouts

	// Обновляем базовую конфигурацию в трекере, но сохраняем информацию об изменённых параметрах
	ch.configTracker.UpdateBaseConfig(newConfig.Weights)

	// После сохранения раскладки, очищаем все временные раскладки, так как они больше не актуальны
	ch.searchResultLayout = nil
	ch.invertedLayout = nil
	ch.isInvertedLayoutActive = false
	ch.bestResults = make([]SimulatedAnnealingResult, 0)

	fmt.Printf("Раскладка '%s' успешно сохранена в файл.\n", layoutToSave.Name)
	return nil
}

// CommandSort сортирует раскладки по возрастанию общей оценки и перезаписывает файл
func (ch *CommandHandler) CommandSort(args string) error {
	if strings.TrimSpace(args) != "" {
		return fmt.Errorf("команда sort не принимает аргументов")
	}

	// Анализируем все раскладки, чтобы получить их оценки
	type ScoredLayout struct {
		Layout Layout
		Score  float64
	}

	var scoredLayouts []ScoredLayout

	for _, layout := range ch.layouts.Layouts {
		analysis := AnalyzeLayout(&layout, ch.config, ch.langData)
		scoredLayouts = append(scoredLayouts, ScoredLayout{
			Layout: layout,
			Score:  analysis.WeightedScore,
		})
	}

	// Сортируем по возрастанию общей оценки
	sort.Slice(scoredLayouts, func(i, j int) bool {
		return scoredLayouts[i].Score < scoredLayouts[j].Score
	})

	// Перезаписываем файл с раскладками в отсортированном порядке
	file, err := os.Create(ch.outputFile)
	if err != nil {
		return fmt.Errorf("ошибка создания файла %s: %v", ch.outputFile, err)
	}
	defer file.Close()

	for i, scoredLayout := range scoredLayouts {
		// Записываем имя раскладки
		if i > 0 {
			// Добавляем пустую строку-разделитель между раскладками (кроме первой)
			if _, err := file.WriteString("\n"); err != nil {
				return fmt.Errorf("ошибка записи разделителя: %v", err)
			}
		}

		// Используем оригинальное имя раскладки
		if _, err := file.WriteString(fmt.Sprintf("%s\n", scoredLayout.Layout.Name)); err != nil {
			return fmt.Errorf("ошибка записи имени раскладки: %v", err)
		}

		// Записываем строки раскладки
		for row := 0; row < 3; row++ {
			line := ""
			for col := 0; col < 10; col++ {
				if col > 0 {
					line += " "
				}
				line += scoredLayout.Layout.Keys[row][col]
				// Добавляем дополнительный пробел между половинками (между 5 и 6 столбцом)
				if col == 4 {
					line += " "
				}
			}
			if _, err := file.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("ошибка записи строки раскладки: %v", err)
			}
		}
	}

	// Перезагружаем файл раскладок, чтобы обновить внутреннее состояние
	newLangData, newConfig, newParsedLayouts, err := LoadAllData(ch.langFile, ch.configFile, ch.outputFile)
	if err != nil {
		return fmt.Errorf("ошибка перезагрузки данных: %v", err)
	}

	// Применяем измененные веса к новой конфигурации
	ch.configTracker.ApplyToConfig(newConfig)

	// Если был указан отдельный файл с усилиями, загружаем матрицу усилий из него
	if ch.effortFile != "" && ch.effortFile != ch.configFile {
		effortMatrix, err := LoadEffortMatrix(ch.effortFile)
		if err != nil {
			fmt.Printf("Предупреждение: не удалось загрузить матрицу усилий из файла %s: %v\n", ch.effortFile, err)
		} else {
			// Обновляем матрицу усилий в конфигурации
			newConfig.EffortMatrix = effortMatrix
		}
	}

	ch.langData = newLangData
	ch.config = newConfig
	ch.layouts = newParsedLayouts

	// Обновляем базовую конфигурацию в трекере, но сохраняем информацию об изменённых параметрах
	ch.configTracker.UpdateBaseConfig(newConfig.Weights)

	// После сортировки файла очищаем временный результат поиска [0],
	// так как нумерация всех раскладок изменилась
	ch.searchResultLayout = nil
	// Also reset the inverted layout active flag if it was active
	if ch.isInvertedLayoutActive {
		ch.isInvertedLayoutActive = false
	}
	ch.bestResults = make([]SimulatedAnnealingResult, 0)

	fmt.Printf("Раскладки успешно отсортированы по возрастанию общей оценки (всего: %d)\n", len(scoredLayouts))
	return nil
}

// CommandDelete удаляет раскладки из файла по номеру или диапазону
func (ch *CommandHandler) CommandDelete(args string) error {
	if strings.TrimSpace(args) == "" {
		return fmt.Errorf("укажите номера или диапазоны раскладок для удаления (например: 1,3-5,0)")
	}

	// Parse the list of indices (e.g., "1,3-5,0")
	indices := make(map[int]bool) // Using map to avoid duplicates
	parts := strings.Split(args, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return fmt.Errorf("некорректный диапазон: %s", part)
			}

			start, err1 := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			end, err2 := strconv.Atoi(strings.TrimSpace(rangeParts[1]))

			if err1 != nil || err2 != nil {
				return fmt.Errorf("некорректные числа в диапазоне: %s", part)
			}

			if start > end {
				start, end = end, start // Swap to make sure start <= end
			}

			for i := start; i <= end; i++ {
				indices[i] = true
			}
		} else {
			num, err := strconv.Atoi(part)
			if err != nil {
				return fmt.Errorf("некорректный номер: %s", part)
			}
			indices[num] = true
		}
	}

	// Check if all indices are valid
	for idx := range indices {
		// Check if index 0 is valid (search result or inverted layout exists)
		if idx == 0 && ch.searchResultLayout == nil && (ch.invertedLayout == nil || !ch.isInvertedLayoutActive) {
			return fmt.Errorf("номер раскладки %d не существует", idx)
		}
		// Check if indices 1+ are within loaded layouts
		if idx > 0 && idx > len(ch.layouts.Layouts) {
			return fmt.Errorf("номер раскладки %d вне диапазона", idx)
		}
		// Check if index is negative
		if idx < 0 {
			return fmt.Errorf("номер раскладки %d вне диапазона", idx)
		}
	}

	// Handle deletion of temporary layout at index 0 (search result or inverted layout)
	if indices[0] {
		ch.searchResultLayout = nil
		ch.invertedLayout = nil
		ch.isInvertedLayoutActive = false
		ch.bestResults = make([]SimulatedAnnealingResult, 0)
	}

	// Create a new layout list without the specified permanent indices (1+)
	newLayouts := []Layout{}
	for i, layout := range ch.layouts.Layouts {
		if !indices[i+1] { // Convert to 1-based index for comparison
			newLayouts = append(newLayouts, layout)
		}
	}

	// Write the new layout list to the file
	file, err := os.Create("layout.txt")
	if err != nil {
		return fmt.Errorf("ошибка создания файла layout.txt: %v", err)
	}
	defer file.Close()

	for i, layout := range newLayouts {
		if _, err := file.WriteString(fmt.Sprintf("%s\n", layout.Name)); err != nil {
			return fmt.Errorf("ошибка записи имени раскладки: %v", err)
		}

		// Записываем строки раскладки
		for row := 0; row < 3; row++ {
			line := ""
			for col := 0; col < 10; col++ {
				if col > 0 {
					line += " "
				}
				line += layout.Keys[row][col]
				// Добавляем дополнительный пробел между половинками (между 5 и 6 столбцом)
				if col == 4 {
					line += " "
				}
			}
			if _, err := file.WriteString(line + "\n"); err != nil {
				return fmt.Errorf("ошибка записи строки раскладки: %v", err)
			}
		}

		// Добавляем пустую строку-разделитель между раскладками, кроме последней
		if i < len(newLayouts)-1 {
			if _, err := file.WriteString("\n"); err != nil {
				return fmt.Errorf("ошибка записи разделителя: %v", err)
			}
		}
	}

	// Update the in-memory layout list
	ch.layouts.Layouts = newLayouts

	// Reload data to update internal state
	newLangData, newConfig, newParsedLayouts, err := LoadAllData(ch.langFile, ch.configFile, ch.layoutFile)
	if err != nil {
		return fmt.Errorf("ошибка перезагрузки данных: %v", err)
	}

	// Применяем измененные веса к новой конфигурации
	ch.configTracker.ApplyToConfig(newConfig)

	// Если был указан отдельный файл с усилиями, загружаем матрицу усилий из него
	if ch.effortFile != "" && ch.effortFile != ch.configFile {
		effortMatrix, err := LoadEffortMatrix(ch.effortFile)
		if err != nil {
			fmt.Printf("Предупреждение: не удалось загрузить матрицу усилий из файла %s: %v\n", ch.effortFile, err)
		} else {
			// Обновляем матрицу усилий в конфигурации
			newConfig.EffortMatrix = effortMatrix
		}
	}

	ch.langData = newLangData
	ch.config = newConfig
	ch.layouts = newParsedLayouts

	// Обновляем базовую конфигурацию в трекере, но сохраняем информацию об изменённых параметрах
	ch.configTracker.UpdateBaseConfig(newConfig.Weights)

	// Print which layouts were deleted
	fmt.Printf("Успешно удалены раскладки: ")
	first := true
	for idx := range indices {
		if !first {
			fmt.Print(", ")
		}
		fmt.Printf("%d", idx)
		first = false
	}
	fmt.Println()

	return nil
}

// CommandRename переименовывает указанную раскладку в файле
func (ch *CommandHandler) CommandRename(args string) error {
	parts := strings.Fields(strings.TrimSpace(args))
	if len(parts) != 2 {
		return fmt.Errorf("используйте: n N имя (где N - номер раскладки, имя - новое имя)")
	}

	numStr := parts[0]
	newName := parts[1]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return fmt.Errorf("некорректный номер раскладки: %v", err)
	}

	if num == 0 {
		// Обработка специального случая: переименование активной раскладки под индексом 0
		if ch.isInvertedLayoutActive && ch.invertedLayout != nil {
			// Переименовываем активную инвертированную раскладку
			oldName := ch.invertedLayout.Name
			ch.invertedLayout.Name = newName
			fmt.Printf("Инвертированная раскладка успешно переименована из '%s' в '%s'\n", oldName, newName)
		} else if ch.searchResultLayout != nil {
			// Переименовываем результат поиска под индексом 0
			oldName := ch.searchResultLayout.Name
			ch.searchResultLayout.Name = newName
			fmt.Printf("Раскладка под индексом 0 успешно переименована из '%s' в '%s'\n", oldName, newName)
		} else {
			return fmt.Errorf("нет активной раскладки под индексом 0 для переименования")
		}
		return nil
	} else if num > 0 && num <= len(ch.layouts.Layouts) {
		// Update the layout name in memory
		oldName := ch.layouts.Layouts[num-1].Name
		ch.layouts.Layouts[num-1].Name = newName

		// Rewrite the whole layout file with updated name
		file, err := os.Create("layout.txt")
		if err != nil {
			return fmt.Errorf("ошибка создания файла layout.txt: %v", err)
		}
		defer file.Close()

		for i, layout := range ch.layouts.Layouts {
			if _, err := file.WriteString(fmt.Sprintf("%s\n", layout.Name)); err != nil {
				return fmt.Errorf("ошибка записи имени раскладки: %v", err)
			}

			// Write layout rows with proper spacing (double space between halves)
			for row := 0; row < 3; row++ {
				line := ""
				for col := 0; col < 10; col++ {
					if col > 0 {
						line += " "
					}
					line += layout.Keys[row][col]
					// Add extra space between halves (after the 5th key/column index 4)
					if col == 4 {
						line += " "
					}
				}
				if _, err := file.WriteString(line + "\n"); err != nil {
					return fmt.Errorf("ошибка записи строки раскладки: %v", err)
				}
			}

			// Add empty line separator between layouts (except after the last one)
			if i < len(ch.layouts.Layouts)-1 {
				if _, err := file.WriteString("\n"); err != nil {
					return fmt.Errorf("ошибка записи разделителя: %v", err)
				}
			}
		}

		fmt.Printf("Раскладка #%d успешно переименована из '%s' в '%s'\n", num, oldName, newName)

		// Reload the layouts to update internal state
		newLangData, newConfig, newLayouts, err := LoadAllData(ch.langFile, ch.configFile, ch.layoutFile)
		if err != nil {
			return fmt.Errorf("ошибка перезагрузки данных: %v", err)
		}

		// Применяем измененные веса к новой конфигурации
		ch.configTracker.ApplyToConfig(newConfig)

		ch.langData = newLangData
		ch.config = newConfig
		ch.layouts = newLayouts

		// Обновляем базовую конфигурацию в трекере, но сохраняем информацию об изменённых параметрах
		ch.configTracker.UpdateBaseConfig(newConfig.Weights)
	} else {
		return fmt.Errorf("номер раскладки вне диапазона")
	}

	return nil
}

// CommandDetailedInfo выводит детализированную информацию о раскладке
func (ch *CommandHandler) CommandDetailedInfo(args string) error {
	if strings.TrimSpace(args) == "" {
		return fmt.Errorf("используйте: i N (где N - номер раскладки)")
	}

	// Парсим номер раскладки
	layoutNum, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		return fmt.Errorf("некорректный номер раскладки: %v", err)
	}

	// Получаем раскладку по номеру
	layout, exists := ch.getLayoutByIndex(layoutNum)
	if !exists {
		return fmt.Errorf("раскладка с номером %d не найдена", layoutNum)
	}

	// Выполняем анализ раскладки
	analysis := AnalyzeLayout(layout, ch.config, ch.langData)
	analysis.LayoutIndex = layoutNum

	// Выводим дополнительные параметры
	fmt.Printf("SKB  = %.2f\n", analysis.BigramAnalysis.SKB)
	fmt.Printf("LSB2 = %.2f\n", analysis.BigramAnalysis.LSB2)
	fmt.Printf("HSB2 = %.2f\n", analysis.BigramAnalysis.HSB2)
	fmt.Printf("FSB2 = %.2f\n", analysis.BigramAnalysis.FSB2)

	// Выводим проверку соотношений
	fmt.Println() // Пустая строка перед проверкой
	fmt.Println("Проверка:")
	fmt.Printf("\nSFB = %.2f\n", analysis.BigramAnalysis.SFB)
	sumSFB := analysis.BigramAnalysis.HVB + analysis.BigramAnalysis.FVB +
	          analysis.BigramAnalysis.HDB + analysis.BigramAnalysis.FDB +
	          analysis.BigramAnalysis.HFB + analysis.BigramAnalysis.SKB
	fmt.Printf("HVB + FVB + HDB + FDB + HFB + SKB = %.2f\n", sumSFB)

	fmt.Printf("\nSHB = %.2f\n", analysis.BigramAnalysis.SHB)
	sumSHB := analysis.BigramAnalysis.SFB + analysis.BigramAnalysis.HSB + analysis.BigramAnalysis.HSB2 +
	          analysis.BigramAnalysis.FSB + analysis.BigramAnalysis.FSB2 +
	          analysis.BigramAnalysis.LSB + analysis.BigramAnalysis.LSB2 +
	          analysis.BigramAnalysis.SRB
	fmt.Printf("SFB + HSB + HSB2 + FSB + FSB2 + LSB + LSB2 + SRB = %.2f\n", sumSHB)

	return nil
}

// CommandLayoutAnalysis выводит подробный анализ конкретной раскладки
func (ch *CommandHandler) CommandLayoutAnalysis(args string) error {
	if strings.TrimSpace(args) == "" {
		return fmt.Errorf("используйте: a N [n] (где N - номер раскладки, n - количество строк в таблицах)")
	}

	// Парсим аргументы
	argParts := strings.Fields(strings.TrimSpace(args))
	if len(argParts) == 0 {
		return fmt.Errorf("используйте: a N [n] (где N - номер раскладки, n - количество строк в таблицах)")
	}

	// Парсим номер раскладки
	layoutNum, err := strconv.Atoi(argParts[0])
	if err != nil {
		return fmt.Errorf("некорректный номер раскладки: %v", err)
	}

	// Парсим опциональное количество строк (по умолчанию 6)
	numRows := 6
	if len(argParts) > 1 {
		n, err := strconv.Atoi(argParts[1])
		if err != nil {
			return fmt.Errorf("некорректное количество строк: %v", err)
		}
		if n <= 0 {
			return fmt.Errorf("количество строк должно быть положительным числом")
		}
		numRows = n
	}

	// Получаем раскладку по номеру
	layout, exists := ch.getLayoutByIndex(layoutNum)
	if !exists {
		return fmt.Errorf("раскладка с номером %d не найдена", layoutNum)
	}

	// Выполняем анализ раскладки
	analysis := AnalyzeLayout(layout, ch.config, ch.langData)
	analysis.LayoutIndex = layoutNum
	// Обрезаем имя для выравнивания колонок
	if len(analysis.LayoutName) > 20 {
		analysis.LayoutName = analysis.LayoutName[:20]
	}

	// Выводим распечатку раскладки (аналогично команде p)
	// Находим максимальную частоту для нормирования
	maxFreq := 0.0
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			key := layout.Keys[row][col]
			if freq, exists := ch.langData.Characters[key]; exists {
				if freq > maxFreq {
					maxFreq = freq
				}
			}
		}
	}

	// Выводим имя раскладки
	if ch.highlightedLayouts[layoutNum] {
		fmt.Printf("\033[38;2;249;226;175m[%d] %s\033[0m\n", layoutNum, layout.Name)
	} else if layoutNum == 0 {
		fmt.Printf("\033[38;2;249;226;175m[%d] %s\033[0m\n", layoutNum, layout.Name) // search result layout gets yellow
	} else {
		fmt.Printf("[%d] %s\n", layoutNum, layout.Name)
	}

	// Выводим строки раскладки с цветным форматированием
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if col == 5 {
				fmt.Print(" ")
			}

			key := layout.Keys[row][col]
			freq := 0.0
			if f, exists := ch.langData.Characters[key]; exists {
				freq = f
			}

			// Нормируем частоту на 100%
			// 0% -> цвет (215,215,215) серый
			// 100% -> цвет (215,0,0) красный
			// остальное -> линейная интерполяция

			var r, g, b int
			if maxFreq > 0 {
				percent := (freq / maxFreq) * 100.0
				// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
				// При 0%: (215,215,215), при 100%: (215,0,0)
				r = 215
				g = 215 - int((percent/100.0)*(215.0))
				b = 215 - int((percent/100.0)*(215.0))
			} else {
				// Если нет данных по частоте - серый
				r, g, b = 215, 215, 215
			}

			// Display with frequency-based color
			fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
		}
		fmt.Println()
	}

	// Пустая строка
	fmt.Println()

	// Выводим строку с информацией по усилиям раскладки (аналогично команде l)
	fmt.Printf(" %-3s %-16s %5s %5s %5s %5s %5s %5s %5s %5s %6s %5s %5s %6s %5s %5s %4s %5s %7s %7s\n",
		"№", "Layout", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "R1", "R2", "R3", "Left", "Right", "HDI", "FDI", "MEP", "Effort", "Score")
	fmt.Println(strings.Repeat("-", 134))
	fmt.Println(FormatAnalysisWithHighlights(analysis))

	// Пустая строка
	fmt.Println()

	// Выводим строку с информацией по биграммам (аналогично команде lb)
	fmt.Printf(" %-3s %-16s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %6s %8s %7s\n",
		"№", "Layout", "SHB", "SFB", "HVB", "FVB", "HDB", "FDB", "HFB", "HSB", "FSB", "LSB", "SRB", "AFI", "AFO", "TIB", "Total", "Score")
	fmt.Println(strings.Repeat("-", 136))
	fmt.Println(FormatBigramAnalysisWithHighlights(analysis))

	// Пустая строка
	fmt.Println()

	// Выводим n самых частых биграмм по различным критериям
	ch.printBigramAnalysisByCriteria(layout, analysis, numRows)

	// Пустая строка
	fmt.Println()

	// Выводим биграммы по типам
	ch.printBigramTypeAnalysis(layout, analysis, numRows)

	return nil
}

// colorizeBigramByFrequency подсвечивает биграмму в зависимости от её частоты и добавляет нормированную частоту
func (ch *CommandHandler) colorizeBigramByFrequency(bigram string, freq float64, maxFreq float64, maxFreqInTable float64) string {
	if maxFreqInTable == 0 {
		// Если максимальная частота в таблице равна 0, используем серый цвет
		colored := fmt.Sprintf("\033[38;2;215;215;215m%s\033[0m", bigram)
		percentage := 0
		if maxFreq > 0 {
			percentage = int(freq * 100.0 / maxFreq)
		}
		percentageStr := fmt.Sprintf("%2d", percentage)
		return fmt.Sprintf("  %s %s", colored, percentageStr)
	}

	// Интерполируем цвет от (215,215,215) (серый, минимальная частота) до (215,0,0) (красный, максимальная частота в таблице)
	relativeFreq := freq / maxFreqInTable

	// Вычисляем цвета: от серого к красному
	r := 215
	g := 215 - int(relativeFreq * 215.0)
	b := 215 - int(relativeFreq * 215.0)

	// Форматируем цветную строку
	colored := fmt.Sprintf("\033[38;2;%d;%d;%dm%s\033[0m", r, g, b, bigram)

	// Вычисляем нормированную частоту как процент от максимальной частоты в языке
	percentage := 0
	if maxFreq > 0 {
		percentage = int(freq * 100.0 / maxFreq)
	}
	// Ограничиваем значение 99 для максимальной частоты, чтобы сохранить форматирование двузначного числа
	if percentage > 99 {
		percentage = 99
	}
	percentageStr := fmt.Sprintf("%2d", percentage)

	return fmt.Sprintf("  %s %s", colored, percentageStr)
}

// printBigramAnalysisByCriteria выводит n самых частых биграмм по различным критериям
func (ch *CommandHandler) printBigramAnalysisByCriteria(layout *Layout, analysis *LayoutAnalysis, numRows int) {
	// Создаём таблицу позиций буквы -> (row, col)
	keyPos := make(map[string][2]int)
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			key := layout.Keys[row][col]
			if key != "" {
				keyPos[key] = [2]int{row, col}
			}
		}
	}

	// Сортируем все биграммы по частоте
	var allBigrams []BigramFreq
	for bigram, freq := range ch.langData.Bigrams {
		allBigrams = append(allBigrams, BigramFreq{Bigram: bigram, Freq: freq})
	}

	// Сортируем по убыванию частоты
	sort.Slice(allBigrams, func(i, j int) bool {
		return allBigrams[i].Freq > allBigrams[j].Freq
	})

	// Подготавливаем биграммы для каждой категории
	var fingerBigrams [8][]BigramFreq
	var rowBigrams [3][]BigramFreq
	var halfBigrams [2][]BigramFreq

	for _, bg := range allBigrams {
		runes := []rune(bg.Bigram)
		if len(runes) != 2 {
			continue
		}

		char1 := string(runes[0])
		char2 := string(runes[1])

		pos1, exists1 := keyPos[char1]
		pos2, exists2 := keyPos[char2]

		if !exists1 || !exists2 {
			continue
		}

		row1, col1 := pos1[0], pos1[1]
		row2, col2 := pos2[0], pos2[1]

		// Проверяем половинки (левая/правая)
		half1_new := getHalf(col1)
		half2_new := getHalf(col2)

		// Биграммы учитываются только если оба символа на одной половинке
		if half1_new != half2_new {
			continue
		}

		// Биграммы, в которых присутствуют символы, нажимаемые соответствующими пальцами
		finger1 := getFingerForKey(row1, col1)
		finger2 := getFingerForKey(row2, col2)

		// Только если хотя бы один символ принадлежит конкретному пальцу
		if finger1 == 0 || finger2 == 0 {
			fingerBigrams[0] = append(fingerBigrams[0], bg)
		}
		if finger1 == 1 || finger2 == 1 {
			fingerBigrams[1] = append(fingerBigrams[1], bg)
		}
		if finger1 == 2 || finger2 == 2 {
			fingerBigrams[2] = append(fingerBigrams[2], bg)
		}
		if finger1 == 3 || finger2 == 3 {
			fingerBigrams[3] = append(fingerBigrams[3], bg)
		}
		if finger1 == 4 || finger2 == 4 {
			fingerBigrams[4] = append(fingerBigrams[4], bg)
		}
		if finger1 == 5 || finger2 == 5 {
			fingerBigrams[5] = append(fingerBigrams[5], bg)
		}
		if finger1 == 6 || finger2 == 6 {
			fingerBigrams[6] = append(fingerBigrams[6], bg)
		}
		if finger1 == 7 || finger2 == 7 {
			fingerBigrams[7] = append(fingerBigrams[7], bg)
		}

		// Биграммы, в которых присутствуют символы, нажимаемые в соответствующих рядах
		if row1 == 0 || row2 == 0 {
			rowBigrams[0] = append(rowBigrams[0], bg)
		}
		if row1 == 1 || row2 == 1 {
			rowBigrams[1] = append(rowBigrams[1], bg)
		}
		if row1 == 2 || row2 == 2 {
			rowBigrams[2] = append(rowBigrams[2], bg)
		}

		// Биграммы, в которых оба символа нажимаются в одной половинке
		if half1_new == 0 && half2_new == 0 {
			halfBigrams[0] = append(halfBigrams[0], bg)  // Left
		} else if half1_new == 1 && half2_new == 1 {
			halfBigrams[1] = append(halfBigrams[1], bg)  // Right
		}
	}

	// Ограничиваем количество биграмм до numRows для каждого типа
	for i := range fingerBigrams {
		if len(fingerBigrams[i]) > numRows {
			fingerBigrams[i] = fingerBigrams[i][:numRows]
		}
	}
	for i := range rowBigrams {
		if len(rowBigrams[i]) > numRows {
			rowBigrams[i] = rowBigrams[i][:numRows]
		}
	}
	for i := range halfBigrams {
		if len(halfBigrams[i]) > numRows {
			halfBigrams[i] = halfBigrams[i][:numRows]
		}
	}

	// Находим максимальную частоту среди всех биграмм для нормировки
	maxFreq := 0.0
	for _, bg := range allBigrams {
		if bg.Freq > maxFreq {
			maxFreq = bg.Freq
		}
	}

	// Находим максимальную частоту среди всех выводимых биграмм для подсветки
	maxFreqInTable := 0.0
	for _, fingerBigramsList := range fingerBigrams {
		for _, bg := range fingerBigramsList {
			if bg.Freq > maxFreqInTable {
				maxFreqInTable = bg.Freq
			}
		}
	}
	for _, rowBigramsList := range rowBigrams {
		for _, bg := range rowBigramsList {
			if bg.Freq > maxFreqInTable {
				maxFreqInTable = bg.Freq
			}
		}
	}
	for _, halfBigramsList := range halfBigrams {
		for _, bg := range halfBigramsList {
			if bg.Freq > maxFreqInTable {
				maxFreqInTable = bg.Freq
			}
		}
	}

	// Выводим заголовок для таблицы биграмм по критериям
	fmt.Printf(" %-3s %-16s  ", "№", "Layout")
	for finger := 0; finger < 8; finger++ {
		fmt.Printf("   F%d   ", finger+1)
	}
	fmt.Printf(" ")
	for row := 0; row < 3; row++ {
		fmt.Printf("   R%d   ", row+1)
	}
	fmt.Printf("   Left   Right")
	fmt.Println()
	fmt.Println(strings.Repeat("-", 127))

	// Подготавливаем и выводим numRows строк биграмм с цветной подсветкой
	for i := 0; i < numRows; i++ {
		if i < 1 {
			fmt.Printf("%-4s %-16s", fmt.Sprintf("[%d]", analysis.LayoutIndex), analysis.LayoutName)
		} else {
			fmt.Printf("%-4s %-16s", "", "")
		}

		// Выводим биграммы для пальцев
		for finger := 0; finger < 8; finger++ {
			if i < len(fingerBigrams[finger]) {
				bigram := fingerBigrams[finger][i].Bigram
				freq := fingerBigrams[finger][i].Freq
				colorizedBigram := ch.colorizeBigramByFrequency(bigram, freq, maxFreq, maxFreqInTable)
				fmt.Printf("%-4s ", colorizedBigram)
			} else {
				fmt.Printf("%-4s ", "")
			}
		}

		fmt.Printf(" ")

		// Выводим биграммы для рядов
		for row := 0; row < 3; row++ {
			if i < len(rowBigrams[row]) {
				bigram := rowBigrams[row][i].Bigram
				freq := rowBigrams[row][i].Freq
				colorizedBigram := ch.colorizeBigramByFrequency(bigram, freq, maxFreq, maxFreqInTable)
				fmt.Printf("%-4s ", colorizedBigram)
			} else {
				fmt.Printf("%-4s ", "")
			}
		}

		fmt.Printf(" ")

		// Выводим биграммы для половин
		for half := 0; half < 2; half++ {
			if i < len(halfBigrams[half]) {
				bigram := halfBigrams[half][i].Bigram
				freq := halfBigrams[half][i].Freq
				colorizedBigram := ch.colorizeBigramByFrequency(bigram, freq, maxFreq, maxFreqInTable)
				fmt.Printf("%-4s ", colorizedBigram)
			} else {
				fmt.Printf("%-4s ", "")
			}
		}

		fmt.Println()
	}
}


// printBigramTypeAnalysis выводит n самых частых биграмм по типам
func (ch *CommandHandler) printBigramTypeAnalysis(layout *Layout, analysis *LayoutAnalysis, numRows int) {
	// Выводим заголовок для таблицы биграмм по типам
	fmt.Printf(" %-3s %-16s %s %-7s %-7s %-7s %-7s %-7s %-7s %-7s %-7s %-7s %-7s %-7s %-7s %-7s\n", "№", "Layout", "  ", "SHB", "SFB", "HVB", "FVB", "HDB", "FDB", "HFB", "HSB", "FSB", "LSB", "SRB", "AFI", "AFO")
	fmt.Println(strings.Repeat("-", 124))

	// Создаём таблицу позиций буквы -> (row, col)
	keyPos := make(map[string][2]int)
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			key := layout.Keys[row][col]
			if key != "" {
				keyPos[key] = [2]int{row, col}
			}
		}
	}

	// Сортируем все биграммы по частоте
	var allBigrams []BigramFreq
	for bigram, freq := range ch.langData.Bigrams {
		allBigrams = append(allBigrams, BigramFreq{Bigram: bigram, Freq: freq})
	}

	// Сортируем по убыванию частоты
	sort.Slice(allBigrams, func(i, j int) bool {
		return allBigrams[i].Freq > allBigrams[j].Freq
	})

	// Подготавливаем биграммы для каждого типа
	var shbBigrams, sfbBigrams, hvbBigrams, fvbBigrams, hdbBigrams, fdbBigrams, hfbBigrams, hsbBigrams, fsbBigrams, lsbBigrams, srbBigrams, afiBigrams, afoBigrams []BigramFreq

	// Подсчитываем биграммы по типам как в calculateBigrams
	for _, bg := range allBigrams {
		runes := []rune(bg.Bigram)
		if len(runes) != 2 {
			continue
		}

		char1 := string(runes[0])
		char2 := string(runes[1])

		pos1, exists1 := keyPos[char1]
		pos2, exists2 := keyPos[char2]

		if !exists1 || !exists2 {
			continue
		}

		row1, col1 := pos1[0], pos1[1]
		row2, col2 := pos2[0], pos2[1]

		// Проверяем половинки (левая/правая)
		half1 := getHalf(col1)
		half2 := getHalf(col2)

		// Проверяем пальцы для обоих символов
		finger1 := getFingerForKey(row1, col1)
		finger2 := getFingerForKey(row2, col2)

		// Вычисляем расстояния
		rowDiff := abs(row1 - row2)
		colDiff := abs(col1 - col2)

		// SHB - Same Hand Bigram (процент биграмм, которые набираются одной рукой)
		if half1 == half2 {
			shbBigrams = append(shbBigrams, bg)
		}

		// SFB - Same Finger Bigrams (процент биграмм, которые набираются одним пальцем)
		if finger1 == finger2 {
			sfbBigrams = append(sfbBigrams, bg)
		}

		// Рассчитываем метрики только если оба символа на одной половинке
		if half1 == half2 {
			// HVB - Half Vertical Bigrams (один палец, одна колонка, соседние ряды, исключая колонки 5 и 6)
			if finger1 == finger2 && col1 == col2 && rowDiff == 1 && col1 != 4 && col1 != 5 {
				hvbBigrams = append(hvbBigrams, bg)
			}

			// FVB - Full Vertical Bigrams (один палец, одна колонка, через ряд, исключая колонки 5 и 6)
			if finger1 == finger2 && col1 == col2 && rowDiff == 2 && col1 != 4 && col1 != 5 {
				fvbBigrams = append(fvbBigrams, bg)
			}

			// HDB - Half Diagonal Bigrams (один палец, соседние колонки и соседние ряды)
			if finger1 == finger2 && rowDiff == 1 && colDiff == 1 {
				hdbBigrams = append(hdbBigrams, bg)
			}

			// FDB - Full Diagonal Bigrams (один палец, соседние колонки через ряд)
			if finger1 == finger2 && rowDiff == 2 && colDiff == 1 {
				fdbBigrams = append(fdbBigrams, bg)
			}

			// HFB - Horizontal Finger Bigrams (один палец, один ряд, соседние колонки)
			if finger1 == finger2 && row1 == row2 && colDiff == 1 {
				hfbBigrams = append(hfbBigrams, bg)
			}

			// SRB - Same Row Bigrams (одна рука, один ряд, исключая колонки 5 и 6)
			if row1 == row2 && col1 != 4 && col1 != 5 && col2 != 4 && col2 != 5 {
				srbBigrams = append(srbBigrams, bg)
			}

			// Определяем специфичные пальцы (2, 3, 6, 7 - индексы 1, 2, 5, 6)
			finger1IsSpecial := (finger1 == 1 || finger1 == 2 || finger1 == 5 || finger1 == 6)
			finger2IsSpecial := (finger2 == 1 || finger2 == 2 || finger2 == 5 || finger2 == 6)

			// HSB - Half Scissors Bigrams: одна рука, разные пальцы, соседние ряды,
			// на НИЖНЕМ из двух рядов находятся пальцы 2, 3, 6 или 7, исключая колонки 5 и 6
			if finger1 != finger2 && rowDiff == 1 && col1 != 4 && col1 != 5 && col2 != 4 && col2 != 5 && half1 == half2 {
				lowerRow := row1
				if row2 > row1 {
					lowerRow = row2
				} // Нижний (с большим номером) из двух рядов
				if lowerRow == row1 && finger1IsSpecial {
					hsbBigrams = append(hsbBigrams, bg)
				} else if lowerRow == row2 && finger2IsSpecial {
					hsbBigrams = append(hsbBigrams, bg)
				}
			}

			// FSB - Full Scissors Bigrams: одна рука, разные пальцы, 1 и 3 ряд,
			// на 3 ряду (row=2 в 0-indexed) находятся пальцы 2, 3, 6 или 7, исключая колонки 5 и 6
			if finger1 != finger2 && rowDiff == 2 && ((row1 == 0 && row2 == 2) || (row1 == 2 && row2 == 0)) && col1 != 4 && col1 != 5 && col2 != 4 && col2 != 5 && half1 == half2 {
				// В 3-м ряду (индекс 2) должен быть палец 2, 3, 6 или 7
				if (row1 == 2 && finger1IsSpecial) || (row2 == 2 && finger2IsSpecial) {
					fsbBigrams = append(fsbBigrams, bg)
				}
			}

			// LSB - Lateral Stretch Bigram (указательный и средний на одной руке через вертикальный ряд, колонки 3-5 или 6-8)
			// Палец 1 = колонка 1 (индекс 1), Палец 2 = колонка 2 (индекс 2), Палец 5 = колонка 7 (индекс 6), Палец 6 = колонка 8 (индекс 7)
			// Это колонки 2-4 или 5-7 (в индексах 0-9)
			isLSBPattern := (col1 == 2 && col2 == 4) || (col1 == 4 && col2 == 2) || (col1 == 5 && col2 == 7) || (col1 == 7 && col2 == 5) // колонки 3-5 или 6-8
			if isLSBPattern {
				lsbBigrams = append(lsbBigrams, bg)
			}

			// AFI - Adjacent Fingers In (соседние клавиши в одном ряду нажимаются по направлению к центру)
			// AFO - Adjacent Fingers Out (соседние клавиши в одном ряду нажимаются по направлению от центра)
			// Центр между колонками 4 и 5 (индексы 4 и 5)
			// AFI: первый символ дальше от центра, второй - ближе к центру
			// AFO: первый символ ближе к центру, второй - дальше от центра
			if row1 == row2 && colDiff == 1 { // одни ряд, соседние колонки
				var centerDist1, centerDist2 float64
				// Расстояние до центра (между колонками 4 и 5)
				centerDist1 = math.Abs(float64(col1) - 4.5)
				centerDist2 = math.Abs(float64(col2) - 4.5)

				if centerDist1 > centerDist2 { // первый символ дальше от центра
					afiBigrams = append(afiBigrams, bg) // движение к центру
				} else if centerDist1 < centerDist2 { // первый символ ближе к центру
					afoBigrams = append(afoBigrams, bg) // движение от центра
				}
			}
		}
	}

	// Ограничиваем количество биграмм до numRows для каждого типа
	bigramLists := [][]BigramFreq{shbBigrams, sfbBigrams, hvbBigrams, fvbBigrams, hdbBigrams, fdbBigrams, hfbBigrams, hsbBigrams, fsbBigrams, lsbBigrams, srbBigrams, afiBigrams, afoBigrams}

	for i := range bigramLists {
		if len(bigramLists[i]) > numRows {
			bigramLists[i] = bigramLists[i][:numRows]
		}
	}

	// Восстанавливаем срезы
	shbBigrams, sfbBigrams, hvbBigrams, fvbBigrams, hdbBigrams, fdbBigrams, hfbBigrams, hsbBigrams, fsbBigrams, lsbBigrams, srbBigrams, afiBigrams, afoBigrams =
		bigramLists[0], bigramLists[1], bigramLists[2], bigramLists[3], bigramLists[4], bigramLists[5], bigramLists[6], bigramLists[7], bigramLists[8], bigramLists[9], bigramLists[10], bigramLists[11], bigramLists[12]

	// Находим максимальную частоту среди всех биграмм для нормировки
	maxFreq := 0.0
	for _, bg := range allBigrams {
		if bg.Freq > maxFreq {
			maxFreq = bg.Freq
		}
	}

	// Находим максимальную частоту среди всех выводимых биграмм для подсветки
	maxFreqInTable := 0.0
	lists := [][]BigramFreq{shbBigrams, sfbBigrams, hvbBigrams, fvbBigrams, hdbBigrams, fdbBigrams, hfbBigrams, hsbBigrams, fsbBigrams, lsbBigrams, srbBigrams, afiBigrams, afoBigrams}
	for _, list := range lists {
		for _, bg := range list {
			if bg.Freq > maxFreqInTable {
				maxFreqInTable = bg.Freq
			}
		}
	}

	// Подготавливаем и выводим numRows строк биграмм для каждого типа с цветовой подсветкой
	for i := 0; i < numRows; i++ {
		if i < 1 {
			fmt.Printf("%-4s %-16s", fmt.Sprintf("[%d]", analysis.LayoutIndex), analysis.LayoutName)
		} else {
			fmt.Printf("%-4s %-16s", "", "")
		}

		// Проверяем и выводим биграммы для каждого типа с цветовой подсветкой
		for _, list := range lists {
			if i < len(list) {
				bigram := list[i].Bigram
				freq := list[i].Freq
				colorizedBigram := ch.colorizeBigramByFrequency(bigram, freq, maxFreq, maxFreqInTable)
				fmt.Printf("%-7s ", colorizedBigram)
			} else {
				fmt.Printf("%-7s ", "")
			}
		}

		// Не выводим Total и Score для этих строк, только биграммы
		fmt.Println()
	}
}

// printHelp выводит справку по командам
func printHelp() {
	helpText := `Доступные команды:
  - p [N,M,L-K]   - Вывести раскладки (все или указанные)
  - l [N,M,L-K]   - Анализ раскладок (все или указанные)
  - lb [N,M,L-K]  - Анализ биграмм (все или указанные)
  - ll [N,M,L-K]  - Анализ раскладок и биграмм (все или указанные)
  - h [N,M,L-K]   - Выделить раскладки N, M и диапазон от L до K желтым цветом, снять выделение с этих раскладок, или, без аргумента снять все выделения
  - a N [n]       - Подробный анализ раскладки N, можно отдельно указать количество строк для вывода в таблицах со статистикой
  - b N [string]  - Визуализация статистики по биграммам для раскладки N, можно указать строку символов для визуализации
  - g [N]         - Поиск оптимальной раскладки, можно указать номер базовой раскладки для поиска
  - gg [N] [file] - Непрерывный поиск оптимальной раскладки, можно указать номер базовой раскладки для поиска и имя файла для сохранения найденных раскладок
  - n N имя       - Переименовать раскладку N в новое имя
  - inv [N]       - Инвертирование активной или указанной раскладке
  - sw [N] ab     - Перестановка двух букв в активной или указанной раскладке
  - s [N] [name]  - Сохранить активную раскладку или указанной раскладки с текущим или указанным именем в файл
  - d [N,M,L-K]   - Удалить раскладки (по номерам или диапазонам)
  - sort          - Сортировать раскладки по возрастанию общей оценки и перезаписать файл
  - c             - Вывести используемые коэффициенты из конфигурационного файла
  - set N value   - Установить коэффициент N в значение value
  - r             - Перезагрузить файл конфигурации и файл с раскладками
  - t             - Вывести тестовую информацию
  - help          - Справка по командам
  - exit/quit/q   - Выход

Описание колонок в таблице с анализом нагрузки:
  F1-F8  - Нагрузка по пальцам
  HDI    - Hand Disbalance Index. Дисбаланс в нагрузке по рукам.
  FDI    - Finger Disbalance Index. Дисбаланс в нагрузке по пальцам.
  Effort - Суммарная нагрузка на пальцы по раскладке.
  Score  - Общая оценка раскладки с учетом нагрузки по пальцам и по биграммам.

Описание колонок в таблице с анализом биграмм:
  SHB    - Same Hand Bigram. Процент биграмм, набираемых одной рукой.
  SFB    - Same Finger Bigrams. Процент биграмм, которые набираются одним пальцем.
  HVB    - Half Vertical Bigrams. Процент биграмм, которые набираются одним пальцем,
           при которых набираемые символы находятся в одной колонке в соседних рядах,
           биграммы в колонках 5 и 6 для данного расчета не учитываются.
  FVB    - Full Vertical Bigrams. Процент биграмм, которые набираются одним пальцем,
           при которых набираемые символы находятся в одной колонке через ряд,
           биграммы в колонках 5 и 6 для данного расчета не учитываются.
  HDB    - Half Diagonal Bigrams. Процент биграмм, набираемых одним пальцем в соседних
           колонках и соседних рядах.
  FDB    - Full Diagonal Bigrams. Процент биграмм, набираемых одним пальцем в соседних
           колонках через ряд.
  HFB    - same Finger Horizontal Bigrams. Процент биграмм, набираемых одним пальцем
           на соседних клавишах по горизонтали, то есть в одном ряду в соседних колонках.
  HSB    - Half Scissors Bigrams. Процент биграмм, набираемых на одной руке, при котором
           разные пальцы находятся в соседних рядах, но при этом на нижнем из двух рядов
           находятся пальцы 2, 3, 6 или 7, биграммы в которых один из символов находится
           в колонке 5 или 6 для данного расчета не учитываются.
  FSB    - Full Scissors Bigrams. Процент биграмм, набираемых на одной руке, при котором
           разные пальцы находятся в 1 и 3 ряду, но при этом на 3 ряду находятся
           пальцы 2, 3, 6 или 7, биграммы в которых один из символов находится
           в колонке 5 или 6 для данного расчета не учитываются.
  LSB    - Lateral Stretch Bigram. Процент биграмм, которые набираются указательным
           и средним пальцем на одной руке через вертикальный ряд, то есть, при которых
           один символ находится в колонке 3, а другой в колонке 5, или один символ
           находится в колонке 6, а другой в колонке 8.
  SRB    - Same Row Bigrams. Процент биграмм, набираемых на одной руке в одном ряду,
           биграммы в которых один из символов находится в колонке 5 или 6 для данного
           расчета не учитываются.
  AFI    - Adjacent Fingers In. Процент биграмм, при которых соседние клавиши в одном
           ряду нажимаются по направлению к центру (движение от внешней клавиши к внутренней).
  AFO    - Adjacent Fingers Out. Процент биграмм, при которых соседние клавиши в одном
           ряду нажимаются по направлению от центра (движение от внутренней клавиши к внешней).
  Total  - Взвешенная сумма с учетом коэффициентов по биграммам.
  Score  - Общая оценка раскладки с учетом нагрузки по пальцам и по биграммам.
`
	fmt.Print(helpText)
}

// saveLayoutToFile сохраняет раскладку в указанный файл
func (ch *CommandHandler) saveLayoutToFile(layout Layout, fileName string) error {
	// Открываем файл для добавления
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла %s для добавления: %v", fileName, err)
	}
	defer file.Close()

	// Записываем пустую строку-разделитель перед новой раскладкой (если файл не пустой)
	// Проверяем, чтобы новые раскладки отделялись от уже имеющихся в файле пустой строкой
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("ошибка получения информации о файле: %v", err)
	}

	// Если файл не пустой, добавляем разделитель перед новой записью
	if fileInfo.Size() > 0 {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("ошибка записи разделителя: %v", err)
		}
	}

	// Создаем имя раскладки в формате "random" и общая оценка
	// Для этого анализируем раскладку и получаем её оценку
	analysis := AnalyzeLayout(&layout, ch.config, ch.langData)
	layoutName := fmt.Sprintf("random %.2f", analysis.WeightedScore)

	// Записываем имя раскладки
	if _, err := file.WriteString(fmt.Sprintf("%s\n", layoutName)); err != nil {
		return fmt.Errorf("ошибка записи имени раскладки: %v", err)
	}

	// Записываем строки раскладки
	for row := 0; row < 3; row++ {
		line := ""
		for col := 0; col < 10; col++ {
			if col > 0 {
				line += " "
			}
			line += layout.Keys[row][col]
			// Добавляем дополнительный пробел между половинками (между 5 и 6 столбцом)
			if col == 4 {
				line += " "
			}
		}
		if _, err := file.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("ошибка записи строки раскладки: %v", err)
		}
	}

	return nil
}

// CommandSwapLetters переставляет две буквы в раскладке и сохраняет результат во временный буфер [0]
func (ch *CommandHandler) CommandSwapLetters(args string) error {
	// Парсим аргументы
	args = strings.TrimSpace(args)
	if args == "" {
		return fmt.Errorf("используйте: sw [N] ab (где N - номер раскладки, ab - две буквы для перестановки)")
	}

	parts := strings.Fields(args)

	var layoutIndex int
	var letters string
	var sourceLayout *Layout

	if len(parts) == 2 {
		// Синтаксис: sw N ab
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("некорректный номер раскладки: %v", err)
		}

		if n <= 0 || n > len(ch.layouts.Layouts) {
			return fmt.Errorf("номер раскладки %d вне диапазона (1-%d)", n, len(ch.layouts.Layouts))
		}

		layoutIndex = n - 1 // преобразуем в индекс с 0
		letters = parts[1]
		sourceLayout = &ch.layouts.Layouts[layoutIndex]
	} else if len(parts) == 1 {
		// Синтаксис: sw ab - работает с текущей активной раскладкой [0], если она есть
		letters = parts[0]

		// Получаем активную раскладку под индексом [0] с учетом всех типов временных раскладок
		activeLayout, exists := ch.getLayoutByIndex(0)
		if !exists || activeLayout == nil {
			return fmt.Errorf("нет активной раскладки в буфере [0] для перестановки букв")
		}

		// Используем текущую активную раскладку
		sourceLayout = activeLayout
	} else {
		return fmt.Errorf("некорректное количество аргументов, используйте: sw [N] ab или sw ab")
	}

	if len([]rune(letters)) != 2 {
		return fmt.Errorf("указанная строка \"%s\" не содержит ровно 2 буквы для перестановки", letters)
	}

	runes := []rune(letters)
	char1 := string(runes[0])
	char2 := string(runes[1])

	// Создаем копию раскладки для модификации
	swappedLayout := Layout{
		Name: sourceLayout.Name + " (sw " + char1 + char2 + ")",
		Keys: [3][10]string{},
	}

	// Копируем ключи
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			swappedLayout.Keys[row][col] = sourceLayout.Keys[row][col]
		}
	}

	// Ищем и меняем местами обе буквы
	foundChar1 := false
	foundChar2 := false

	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if swappedLayout.Keys[row][col] == char1 {
				swappedLayout.Keys[row][col] = char2
				foundChar1 = true
			} else if swappedLayout.Keys[row][col] == char2 {
				swappedLayout.Keys[row][col] = char1
				foundChar2 = true
			}
		}
	}

	if !foundChar1 || !foundChar2 {
		return fmt.Errorf("не удалось найти обе буквы \"%s\" и/или \"%s\" в раскладке", char1, char2)
	}

	// Выводим результат
	fmt.Printf("\n%s\n", swappedLayout.Name)
	fmt.Println(strings.Repeat("-", len(swappedLayout.Name)))

	// Определяем максимальную частоту для нормализации цвета
	maxFreq := 0.0
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			key := swappedLayout.Keys[row][col]
			if freq, exists := ch.langData.Characters[key]; exists {
				if freq > maxFreq {
					maxFreq = freq
				}
			}
		}
	}

	// Выводим строки раскладки с цветовым форматированием
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if col == 5 {
				fmt.Print(" ")
			}

			key := swappedLayout.Keys[row][col]
			freq := 0.0
			if f, exists := ch.langData.Characters[key]; exists {
				freq = f
			}

			// Нормируем частоту на 100%
			// 0% -> цвет (215,215,215) серый
			// 100% -> цвет (215,0,0) красный
			// остальное -> линейная интерполяция

			var r, g, b int
			if maxFreq > 0 {
				percent := (freq / maxFreq) * 100.0
				// Интерполируем цвет: серый (215,215,215) -> красный (215,0,0)
				// При 0%: (215,215,215), при 100%: (215,0,0)
				r = 215
				g = 215 - int((percent/100.0)*(215.0))
				b = 215 - int((percent/100.0)*(215.0))
			} else {
				// Если нет данных по частоте - серый
				r, g, b = 215, 215, 215
			}

			// Display with frequency-based color
			fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
		}
		fmt.Println()
	}
	fmt.Println()

	// Сохраняем результат во временный буфер [0]
	ch.searchResultLayout = &swappedLayout

	// Reset the inverted layout active flag since we now have a regular swapped layout in [0]
	ch.isInvertedLayoutActive = false
	// Also clear the inverted layout since it's no longer active
	ch.invertedLayout = nil

	return nil
	return nil
}

// CommandBigramLetter выводит визуализацию частот биграмм для заданной буквы в раскладке
func (ch *CommandHandler) CommandBigramLetter(args string) error {
	// Получаем строку аргументов
	argParts := strings.Fields(args)

	var layoutIndex int
	var letters string

	// Проверяем, есть ли первый аргумент - номер раскладки
	if len(argParts) > 0 {
		// Пытаемся распознать номер раскладки
		if idx, err := strconv.Atoi(argParts[0]); err == nil && idx >= 0 {  // Изменили условие на idx >= 0
			// Используем пользовательскую индексацию (начиная с 0), преобразуем в системную
			layoutIndex = idx
			// Если есть второй аргумент, это строка букв
			if len(argParts) > 1 {
				letters = argParts[1]
			} else {
				// Если нет строки букв, выводим для всех букв в алфавитном порядке
				// Собираем все уникальные буквы из указанной раскладки
				allLetters := make(map[rune]bool)
				layout, found := ch.getLayoutByIndex(layoutIndex)
				if found {
					for row := 0; row < 3; row++ {
						for col := 0; col < 10; col++ {
							key := layout.Keys[row][col]
							for _, char := range key {
								allLetters[char] = true
							}
						}
					}
				}

				// Преобразуем в слайс и сортируем
				var sortedLetters []rune
				for letter := range allLetters {
					sortedLetters = append(sortedLetters, letter)
				}
				sort.Slice(sortedLetters, func(i, j int) bool {
					return sortedLetters[i] < sortedLetters[j]
				})

				letters = string(sortedLetters)
			}
		} else {
			// Первый аргумент - строка букв, используем первую раскладку (индекс 0, т.е. layout 1)
			layoutIndex = 0
			letters = argParts[0]
		}
	} else {
		// Если аргументов нет, используем первую раскладку и все буквы в алфавитном порядке
		layoutIndex = 0
		// Собираем все уникальные буквы из первой раскладки
		allLetters := make(map[rune]bool)
		layout, found := ch.getLayoutByIndex(layoutIndex)
		if found {
			for row := 0; row < 3; row++ {
				for col := 0; col < 10; col++ {
					key := layout.Keys[row][col]
					for _, char := range key {
						allLetters[char] = true
					}
				}
			}
		}

		// Преобразуем в слайс и сортируем
		var sortedLetters []rune
		for letter := range allLetters {
			sortedLetters = append(sortedLetters, letter)
		}
		sort.Slice(sortedLetters, func(i, j int) bool {
			return sortedLetters[i] < sortedLetters[j]
		})

		letters = string(sortedLetters)
	}

	// Проверяем, что раскладка с указанным индексом существует
	layout, layoutFound := ch.getLayoutByIndex(layoutIndex)
	if !layoutFound {
		return fmt.Errorf("раскладка с индексом %d не найдена", layoutIndex) // Показываем индекс как есть
	}

	// Обрабатываем каждую букву из строки
	lettersProcessed := 0 // Счетчик найденных букв в раскладке

	// Собираем все буквы, которые нужно обработать
	var validLetters []rune
	for _, letter := range letters {
		// Проверяем, содержится ли буква в раскладке (регистронезависимо)
		letterFound := false
		letterStr := string(letter)
		letterLower := strings.ToLower(letterStr)

		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				key := layout.Keys[row][col]
				keyLower := strings.ToLower(key)
				if key == letterStr || keyLower == letterLower {
					letterFound = true
					break
				}
			}
			if letterFound {
				break
			}
		}

		if letterFound {
			validLetters = append(validLetters, letter)
		} else {
			fmt.Printf("Буква '%c' не найдена в раскладке %d\n", letter, layoutIndex)
		}
	}

	// Выводим информацию для каждой найденной буквы
	for _, letter := range validLetters {
		// Печатаем строку разделителя ДО вывода для каждой буквы
		fmt.Printf("\n--[ \033[38;2;0;255;255m%c\033[0m ]%s\n", letter, strings.Repeat("-", 60))

		err := ch.displayBigramForLetter(layout, string(letter))
		if err != nil {
			return err
		}
		lettersProcessed++
	}

	// Если ни одна буква не была найдена в раскладке
	if lettersProcessed == 0 && letters != "" {
		fmt.Printf("Ни одна из указанных букв не найдена в раскладке %d\n", layoutIndex+1)
	} else if lettersProcessed > 0 {
		// Выводим завершающий разделитель после всех букв
		fmt.Println(strings.Repeat("-", 67))
	}

	return nil
}

// displayBigramForLetter выводит визуализацию биграмм для конкретной буквы
func (ch *CommandHandler) displayBigramForLetter(layout *Layout, letter string) error {
	// Находим позицию буквы в раскладке (регистронезависимо)
	letterRow, letterCol := -1, -1
	letterLower := strings.ToLower(letter)

	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			key := layout.Keys[row][col]
			keyLower := strings.ToLower(key)
			if key == letter || keyLower == letterLower {
				letterRow, letterCol = row, col
				break
			}
		}
		if letterRow != -1 {
			break
		}
	}

	if letterRow == -1 {
		// Буква не найдена в раскладке, пропускаем
		return nil
	}

	// Определяем, какая половинка клавиатуры содержит букву (левая или правая)
	isLeftSide := letterCol < 5

	// Выводим первый блок (биграммы, где буква на первом месте)
	ch.printBigramBlock(layout, letter, isLeftSide, true)

	fmt.Println() // Пустая строка между блоками

	// Выводим второй блок (биграммы, где буква на втором месте)
	ch.printBigramBlock(layout, letter, isLeftSide, false)

	return nil
}

// printBigramBlock выводит блок биграмм для конкретной буквы
func (ch *CommandHandler) printBigramBlock(layout *Layout, letter string, isLeftSide bool, isFirstPosition bool) {
	// Сначала определим максимальную частоту среди всех биграмм в языковом файле
	maxFreqInLanguage := 0.0
	for _, freq := range ch.langData.Bigrams {
		if freq > maxFreqInLanguage {
			maxFreqInLanguage = freq
		}
	}

	// Если максимальная частота 0, устанавливаем какую-то минимальную для избежания деления на 0
	if maxFreqInLanguage == 0 {
		maxFreqInLanguage = 1
	}

	// Для каждого ряда формируем 3 буферные строки
	for row := 0; row < 3; row++ {
		// Первая буферная строка: раскладка с цветами + 4 пробела + 5 биграмм (нормировка на макс. частоту в языке) + 4 пробела + 5 биграмм (нормировка на частоту на половинке)

		// Выводим раскладку для текущего ряда
		for col := 0; col < 10; col++ {
			if col == 5 {
				fmt.Print(" ") // Пробел между половинками
			}

			key := layout.Keys[row][col]

			// Определяем цвет
			var r, g, b int

			if key == letter {
				// Cyan цвет для указанной буквы
				r, g, b = 0, 255, 255
			} else if (isLeftSide && col < 5) || (!isLeftSide && col >= 5) {
				// Та же половинка, что и буква - цвет от красного к серому в зависимости от частоты биграммы с участием заданной буквы
				var bigramFreq float64
				var bigram string
				if isFirstPosition {
					bigram = letter + key  // Например, "хы" при целевой "х" и текущей "ы"
				} else {
					bigram = key + letter  // Например, "ых" при целевой "х" и текущей "ы"
				}

				// Находим максимальную частоту биграммы с участием заданной буквы на той же половинке
				maxFreqForLetter := 0.0
				for r_idx := 0; r_idx < 3; r_idx++ {
					for c_idx := 0; c_idx < 10; c_idx++ {
						sideKey := layout.Keys[r_idx][c_idx]
						// Проверяем, находится ли буква на той же половинке
						onSameSideCheck := (isLeftSide && c_idx < 5) || (!isLeftSide && c_idx >= 5)

						if onSameSideCheck && sideKey != "" {
							var sideBigram string
							if isFirstPosition {
								sideBigram = letter + sideKey  // "х" + sideKey
							} else {
								sideBigram = sideKey + letter  // sideKey + "х"
							}

							if freq, exists := ch.langData.Bigrams[sideBigram]; exists && freq > maxFreqForLetter {
								maxFreqForLetter = freq
							} else if !exists {
								// Если не найдена, ищем регистронезависимо
								sideBigramLower := strings.ToLower(sideBigram)
								if freq, exists := ch.langData.Bigrams[sideBigramLower]; exists && freq > maxFreqForLetter {
									maxFreqForLetter = freq
								}
							}
						}
					}
				}

				// Получаем частоту биграммы для текущей буквы (регистронезависимо)
				bigramFreq = 0 // по умолчанию 0
				if freq, exists := ch.langData.Bigrams[bigram]; exists {
					bigramFreq = freq
				} else {
					// Если не найдена, ищем регистронезависимо
					bigramLower := strings.ToLower(bigram)
					if freq, exists := ch.langData.Bigrams[bigramLower]; exists {
						bigramFreq = freq
					}
				}

				// Нормируем частоту на максимальное значение частоты биграммы в языковом файле
				percent := 0.0
				if maxFreqInLanguage > 0 {
					percent = (bigramFreq / maxFreqInLanguage) * 100.0
				} else {
					percent = 0 // если максимальная частота 0, то и текущая 0
				}
				if percent > 100 {
					percent = 100
				}

				// Интерполируем цвет: красный (215,0,0) -> серый (215,215,215)
				r = 215
				g = int((1 - (percent/100.0)) * 215.0)
				b = int((1 - (percent/100.0)) * 215.0)
			} else {
				// Противоположная половинка - серый цвет
				r, g, b = 215, 215, 215
			}

			if key == "" {
				fmt.Print("  ") // Пустое место для пустой клавиши
			} else {
				fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m ", r, g, b, key)
			}
		}

		// Добавляем 4 пробела
		fmt.Print("   ")

		// Выводим 5 биграмм с нормировкой на максимальную частоту в языке
		count := 0
		for col := 0; col < 10 && count < 5; col++ {
			key := layout.Keys[row][col]

			// Проверяем, находится ли буква на той же половинке
			onSameSide := (isLeftSide && col < 5) || (!isLeftSide && col >= 5)

			// Для каждой позиции на той же половинке выводим биграмму
			if onSameSide {
				count++

				var bigram string
				if isFirstPosition {
					bigram = letter + key
				} else {
					bigram = key + letter
				}

				// Проверяем, есть ли такая биграмма в данных (регистронезависимо)
				freq, exists := ch.langData.Bigrams[bigram]
				if !exists {
					// Если не найдена, ищем регистронезависимо
					bigramLower := strings.ToLower(bigram)
					freq, exists = ch.langData.Bigrams[bigramLower]
					if !exists {
						freq = 0 // Если биграмма отсутствует, используем нулевое значение
					}
				}

				// Нормируем частоту для отображения
				percent := 0.0
				if maxFreqInLanguage > 0 {
					percent = (freq / maxFreqInLanguage) * 100.0
				}
				displayPercent := int(percent)
				if displayPercent > 99 {
					displayPercent = 99 // Для максимальной частоты используем 99
				}

				// Определяем цвет биграммы (от красного к серому)
				bigramPercent := 0.0
				if maxFreqInLanguage > 0 {
					bigramPercent = (freq / maxFreqInLanguage) * 100.0
				}
				var br, bg_color, bb int
				if bigramPercent > 100 {
					bigramPercent = 100
				}

				// Интерполируем цвет: красный (215,0,0) -> серый (215,215,215)
				br = 215
				bg_color = int((1 - (bigramPercent / 100.0)) * 215.0)
				bb = int((1 - (bigramPercent / 100.0)) * 215.0)

				// Выводим биграмму и частоту в квадратных скобках: [биграмма частота]
				fmt.Printf("\033[38;2;%d;%d;%dm[\033[0m", 215, 215, 215) // Серый цвет для скобок
				fmt.Printf("\033[38;2;%d;%d;%dm%s\033[0m", br, bg_color, bb, bigram) // Цветная биграмма
				fmt.Printf("\033[38;2;215;215;215m %2d]\033[0m  ", displayPercent) // Серый цвет для частоты и 2 пробела
			}
		}

		fmt.Println()
	}
}
