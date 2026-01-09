package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// LoadLanguageData загружает данные о языке из JSON файла
func LoadLanguageData(filename string) (*LanguageData, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла языка: %w", err)
	}

	var langData LanguageData
	err = json.Unmarshal(data, &langData)
	if err != nil {
		return nil, fmt.Errorf("ошибка при парсинге JSON: %w", err)
	}

	return &langData, nil
}

// LoadKeyboardConfig загружает конфигурацию клавиатуры из текстового файла
func LoadKeyboardConfig(filename string) (*KeyboardConfig, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла конфигурации: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(file)), "\n")
	config := &KeyboardConfig{}

	if err := parseEffortMatrix(lines, config); err != nil {
		return nil, err
	}

	if err := parseMaxFingerEfforts(lines, config); err != nil {
		return nil, err
	}

	if err := parseFixedPositions(lines, config); err != nil {
		return nil, err
	}

	if err := parseWeights(lines, config); err != nil {
		return nil, err
	}

	return config, nil
}

// parseEffortMatrix парсит матрицу усилий из конфигурации
func parseEffortMatrix(lines []string, config *KeyboardConfig) error {
	rowIndex := 0
	for i := 0; i < len(lines) && rowIndex < 3; i++ {
		line := lines[i]

		// Удаляем комментарии (все после #)
		commentIdx := strings.Index(line, "#")
		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)

		// Пропускаем пустые строки и комментарии
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 10 {
			return fmt.Errorf("строка %d имеет менее 10 значений усилий", rowIndex+1)
		}

		for col := 0; col < 10; col++ {
			val, err := strconv.ParseFloat(parts[col], 64)
			if err != nil {
				return fmt.Errorf("ошибка парсинга усилия [%d][%d]: %w", rowIndex, col, err)
			}
			config.EffortMatrix[rowIndex][col] = val
		}

		rowIndex++
	}

	if rowIndex < 3 {
		return fmt.Errorf("недостаточно строк для матрицы усилий")
	}

	return nil
}

// parseMaxFingerEfforts парсит максимальные значения усилия для каждого пальца
func parseMaxFingerEfforts(lines []string, config *KeyboardConfig) error {
	// Начинаем поиск с начала списка строк, пропуская комментарии и пустые строки
	startIdx := 0
	for startIdx < len(lines) {
		line := lines[startIdx]

		// Удаляем комментарии (все после #)
		commentIdx := strings.Index(line, "#")
		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)

		// Пропускаем пустые строки и комментарии
		if line == "" {
			startIdx++
			continue
		}

		// Проверяем, содержит ли строка 8 числовых значений (максимальные усилия для пальцев)
		parts := strings.Fields(line)
		if len(parts) >= 8 {
			// Проверяем, являются ли первые 8 элементов числами
			allNumbers := true
			for i := 0; i < 8 && i < len(parts); i++ {
				if _, err := strconv.ParseFloat(parts[i], 64); err != nil {
					allNumbers = false
					break
				}
			}

			if allNumbers {
				break // Нашли подходящую строку
			}
		}

		startIdx++
	}

	if startIdx >= len(lines) {
		return fmt.Errorf("не найдена строка с максимальными значениями усилия для пальцев")
	}

	parts := strings.Fields(lines[startIdx])
	if len(parts) < 8 {
		return fmt.Errorf("строка с максимальными значениями усилия для пальцев имеет менее 8 значений")
	}

	for i := 0; i < 8; i++ {
		val, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return fmt.Errorf("ошибка парсинга максимального усилия для пальца %d: %w", i+1, err)
		}
		config.MaxFingerEfforts[i] = val
	}

	// Теперь ищем следующую строку с штрафами за превышение для пальцев
	startIdx++
	for startIdx < len(lines) {
		line := lines[startIdx]

		// Удаляем комментарии (все после #)
		commentIdx := strings.Index(line, "#")
		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)

		// Пропускаем пустые строки и комментарии
		if line == "" {
			startIdx++
			continue
		}

		// Проверяем, содержит ли строка 8 числовых значений (штрафы за превышение)
		parts := strings.Fields(line)
		if len(parts) >= 8 {
			// Проверяем, являются ли первые 8 элементов числами
			allNumbers := true
			for i := 0; i < 8 && i < len(parts); i++ {
				if _, err := strconv.ParseFloat(parts[i], 64); err != nil {
					allNumbers = false
					break
				}
			}

			if allNumbers {
				break // Нашли подходящую строку
			}
		}

		startIdx++
	}

	if startIdx >= len(lines) {
		return fmt.Errorf("не найдена строка с штрафами за превышение максимальной нагрузки на пальцы")
	}

	parts = strings.Fields(lines[startIdx])
	if len(parts) < 8 {
		return fmt.Errorf("строка с штрафами за превышение максимальной нагрузки имеет менее 8 значений")
	}

	for i := 0; i < 8; i++ {
		val, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return fmt.Errorf("ошибка парсинга штрафа за превышение для пальца %d: %w", i+1, err)
		}
		config.FingerEffortPenalties[i] = val
	}

	return nil
}

// parseFixedPositions парсит матрицу фиксированных позиций
func parseFixedPositions(lines []string, config *KeyboardConfig) error {
	// Найдем позицию после строки с максимальными усилиями для пальцев
	// Ищем первую строку, начинающуюся с '. или 'x' - это начало матрицы фиксированных позиций
	startIdx := 0
	foundFixedPositionsStart := false

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]

		// Удаляем комментарии (все после #)
		commentIdx := strings.Index(line, "#")
		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)

		// Пропускаем пустые строки и комментарии
		if line == "" {
			continue
		}

		// Проверяем, может ли эта строка быть началом матрицы фиксированных позиций
		// Это строка, в которой есть '.' или 'x' как первые значимые символы
		parts := strings.Fields(line)
		if len(parts) == 10 && (parts[0] == "." || parts[0] == "x") {
			startIdx = i
			foundFixedPositionsStart = true
			break
		}
	}

	if !foundFixedPositionsStart {
		return fmt.Errorf("не найдена матрица фиксированных позиций")
	}

	for row := 0; row < 3; row++ {
		if startIdx+row >= len(lines) {
			return fmt.Errorf("недостаточно строк для матрицы позиций")
		}

		// Обрабатываем строку с фиксированными позициями, учитывая возможные комментарии
		line := lines[startIdx+row]

		// Удаляем комментарии (все после #)
		commentIdx := strings.Index(line, "#")
		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)

		parts := strings.Fields(line)
		if len(parts) < 10 {
			return fmt.Errorf("строка позиций %d имеет менее 10 значений", row+1)
		}

		for col := 0; col < 10; col++ {
			config.FixedPositions[row][col] = parts[col]
		}
	}

	return nil
}

// parseWeights парсит коэффициенты весов
func parseWeights(lines []string, config *KeyboardConfig) error {
	config.Weights = WeightConfig{
		Effort:         0.3,
		HandSwitch:     0.2,
		SameFinger:     0.15,
		SameFingerJump: 0.1,
		Inroll:         0.15,
		Outroll:        0.1,
		// Нормирующие коэффициенты для биграмм по умолчанию
		SHB:             0.1,
		SFB:             0.1,
		HVB:             0.1,
		FVB:             0.1,
		HDB:             0.1,
		FDB:             0.1,
		HFB:             0.1,
		HSB:             0.1,
		FSB:             0.1,
		LSB:             0.1,
		SRB:             0.1,
		AFI:             0.1,
		AFO:             0.1,
		HDI:             0.1,
		FDI:             0.1,
		D18:             0.1,
		D27:             0.1,
		D36:             0.1,
		D45:             0.1,
		TotalEffortNorm: 0.01,
		HSBStrictMode:   1,  // Strict mode ON by default
		FSBStrictMode:   1,  // Strict mode ON by default
	}

	// Создаем флаги для отслеживания, были ли прочитаны все параметры
	flags := make(map[string]bool)
	allParams := []string{
		"effort", "hand_switch", "same_finger", "same_finger_jump", "inroll", "outroll",
		"SHB", "SFB", "HVB", "FVB", "HDB", "FDB", "HFB", "HSB", "FSB", "LSB", "SRB", "AFI", "AFO", "HDI", "FDI", "D18", "D27", "D36", "D45", "HSB_strict_mode", "FSB_strict_mode", "LSB_strict_mode", "total_effort_norm",
	}

	totalParams := len(allParams)
	loadedParams := 0

	// Ищем строки с весами
	for _, line := range lines {
		// Удаляем комментарии (все после #)
		commentIdx := strings.Index(line, "#")
		if commentIdx != -1 {
			line = line[:commentIdx]
		}

		line = strings.TrimSpace(line)

		// Пропускаем строки, начинающиеся с # (уже удалены, но на всякий случай проверяем пустые строки)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Проверяем, является ли строка индивидуальным коэффициентом для биграммы
		if isBigramIndividualCoeffLine(line) {
			coeffs, err := parseBigramIndividualCoeffLine(line)
			if err == nil {
				config.BigramIndividualCoeffs = append(config.BigramIndividualCoeffs, coeffs...)
			}
			continue
		}

		if loadedParams >= totalParams {
			// Все параметры уже загружены, прекращаем обработку
			break
		}

		if strings.HasPrefix(line, "effort=") && !flags["effort"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "effort="), 64)
			config.Weights.Effort = val
			flags["effort"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "hand_switch=") && !flags["hand_switch"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "hand_switch="), 64)
			config.Weights.HandSwitch = val
			flags["hand_switch"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "same_finger=") && !flags["same_finger"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "same_finger="), 64)
			config.Weights.SameFinger = val
			flags["same_finger"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "same_finger_jump=") && !flags["same_finger_jump"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "same_finger_jump="), 64)
			config.Weights.SameFingerJump = val
			flags["same_finger_jump"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "inroll=") && !flags["inroll"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "inroll="), 64)
			config.Weights.Inroll = val
			flags["inroll"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "outroll=") && !flags["outroll"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "outroll="), 64)
			config.Weights.Outroll = val
			flags["outroll"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "SHB=") && !flags["SHB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "SHB="), 64)
			config.Weights.SHB = val
			flags["SHB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "SFB=") && !flags["SFB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "SFB="), 64)
			config.Weights.SFB = val
			flags["SFB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "HVB=") && !flags["HVB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "HVB="), 64)
			config.Weights.HVB = val
			flags["HVB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "FVB=") && !flags["FVB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "FVB="), 64)
			config.Weights.FVB = val
			flags["FVB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "HDB=") && !flags["HDB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "HDB="), 64)
			config.Weights.HDB = val
			flags["HDB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "FDB=") && !flags["FDB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "FDB="), 64)
			config.Weights.FDB = val
			flags["FDB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "HFB=") && !flags["HFB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "HFB="), 64)
			config.Weights.HFB = val
			flags["HFB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "HSB=") && !flags["HSB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "HSB="), 64)
			config.Weights.HSB = val
			flags["HSB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "FSB=") && !flags["FSB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "FSB="), 64)
			config.Weights.FSB = val
			flags["FSB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "LSB=") && !flags["LSB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "LSB="), 64)
			config.Weights.LSB = val
			flags["LSB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "SRB=") && !flags["SRB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "SRB="), 64)
			config.Weights.SRB = val
			flags["SRB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "HDI=") && !flags["HDI"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "HDI="), 64)
			config.Weights.HDI = val
			flags["HDI"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "FDI=") && !flags["FDI"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "FDI="), 64)
			config.Weights.FDI = val
			flags["FDI"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "D18=") && !flags["D18"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "D18="), 64)
			config.Weights.D18 = val
			flags["D18"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "D27=") && !flags["D27"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "D27="), 64)
			config.Weights.D27 = val
			flags["D27"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "D36=") && !flags["D36"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "D36="), 64)
			config.Weights.D36 = val
			flags["D36"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "D45=") && !flags["D45"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "D45="), 64)
			config.Weights.D45 = val
			flags["D45"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "HSB_strict_mode=") && !flags["HSB_strict_mode"] {
			val, _ := strconv.Atoi(strings.TrimPrefix(line, "HSB_strict_mode="))
			config.Weights.HSBStrictMode = val
			flags["HSB_strict_mode"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "FSB_strict_mode=") && !flags["FSB_strict_mode"] {
			val, _ := strconv.Atoi(strings.TrimPrefix(line, "FSB_strict_mode="))
			config.Weights.FSBStrictMode = val
			flags["FSB_strict_mode"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "LSB_strict_mode=") && !flags["LSB_strict_mode"] {
			val, _ := strconv.Atoi(strings.TrimPrefix(line, "LSB_strict_mode="))
			config.Weights.LSBStrictMode = val
			flags["LSB_strict_mode"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "SRB=") && !flags["SRB"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "SRB="), 64)
			config.Weights.SRB = val
			flags["SRB"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "AFI=") && !flags["AFI"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "AFI="), 64)
			config.Weights.AFI = val
			flags["AFI"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "AFO=") && !flags["AFO"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "AFO="), 64)
			config.Weights.AFO = val
			flags["AFO"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "total_effort_norm=") && !flags["total_effort_norm"] {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "total_effort_norm="), 64)
			config.Weights.TotalEffortNorm = val
			flags["total_effort_norm"] = true
			loadedParams++
		} else if strings.HasPrefix(line, "MR1=") {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "MR1="), 64)
			config.MaxRowEfforts[0] = val
		} else if strings.HasPrefix(line, "MR2=") {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "MR2="), 64)
			config.MaxRowEfforts[1] = val
		} else if strings.HasPrefix(line, "MR3=") {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "MR3="), 64)
			config.MaxRowEfforts[2] = val
		} else if strings.HasPrefix(line, "PR1=") {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "PR1="), 64)
			config.RowEffortPenalties[0] = val
		} else if strings.HasPrefix(line, "PR2=") {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "PR2="), 64)
			config.RowEffortPenalties[1] = val
		} else if strings.HasPrefix(line, "PR3=") {
			val, _ := strconv.ParseFloat(strings.TrimPrefix(line, "PR3="), 64)
			config.RowEffortPenalties[2] = val
		}
	}

	// Синхронизируем значения с WeightConfig
	config.Weights.MaxRowEffort1 = config.MaxRowEfforts[0]
	config.Weights.MaxRowEffort2 = config.MaxRowEfforts[1]
	config.Weights.MaxRowEffort3 = config.MaxRowEfforts[2]
	config.Weights.RowPenalty1 = config.RowEffortPenalties[0]
	config.Weights.RowPenalty2 = config.RowEffortPenalties[1]
	config.Weights.RowPenalty3 = config.RowEffortPenalties[2]

	return nil
}

// isBigramIndividualCoeffLine проверяет, является ли строка индивидуальным коэффициентом для биграммы
func isBigramIndividualCoeffLine(line string) bool {
	// Проверяем, начинается ли строка с числа (возможно с минусом и точкой), за которым следует двоеточие
	trimmed := strings.TrimSpace(line)

	// Ищем двоеточие
	colonIdx := strings.Index(trimmed, ":")
	if colonIdx == -1 {
		return false
	}

	// Проверяем, что перед двоеточием идет число
	coeffStr := strings.TrimSpace(trimmed[:colonIdx])
	_, err := strconv.ParseFloat(coeffStr, 64)
	return err == nil
}

// parseBigramIndividualCoeffLine парсит строку с индивидуальными коэффициентами для биграмм
func parseBigramIndividualCoeffLine(line string) ([]BigramIndividualCoeff, error) {
	// Ищем двоеточие
	colonIdx := strings.Index(line, ":")
	if colonIdx == -1 {
		return nil, fmt.Errorf("не найдено двоеточие в строке: %s", line)
	}

	// Извлекаем коэффициент
	coeffStr := strings.TrimSpace(line[:colonIdx])
	coeff, err := strconv.ParseFloat(coeffStr, 64)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга коэффициента: %v", err)
	}

	// Извлекаем биграммы
	bigramsStr := strings.TrimSpace(line[colonIdx+1:])

	// Разбиваем строку с биграммами по пробелам
	parts := strings.Fields(bigramsStr)

	var coeffs []BigramIndividualCoeff

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Парсим биграмму в формате "12-13"
		bigramParts := strings.Split(part, "-")
		if len(bigramParts) != 2 {
			continue // Пропускаем неправильные биграммы
		}

		pos1, err1 := strconv.Atoi(bigramParts[0])
		pos2, err2 := strconv.Atoi(bigramParts[1])

		// Проверяем, что номера позиций валидны (1-30)
		if err1 != nil || err2 != nil || pos1 < 1 || pos1 > 30 || pos2 < 1 || pos2 > 30 {
			continue // Пропускаем неправильные биграммы
		}

		// Преобразуем в индекс (0-29)
		coeffs = append(coeffs, BigramIndividualCoeff{
			Pos1: pos1 - 1,
			Pos2: pos2 - 1,
			Coeff: coeff,
		})
	}

	return coeffs, nil
}

// LoadLayouts загружает раскладки из текстового файла
func LoadLayouts(filename string) (*ParsedLayouts, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла раскладок: %w", err)
	}

	layouts := &ParsedLayouts{
		Layouts: []Layout{},
		FileHeaderComments: []string{},  // Инициализируем пустым массивом
	}

	lines := strings.Split(string(file), "\n")
	var currentLayout *Layout
	var rowCount int
	var preLayoutComments []string  // Комментарии перед раскладкой
	var inHeader = true  // Флаг, указывающий, что мы все еще в заголовке файла

	for _, line := range lines {
		originalLine := line
		trimmedLine := strings.TrimSpace(line)

		// Обрабатываем комментарии
		if strings.HasPrefix(trimmedLine, "#") {
			if currentLayout == nil {
				// Если раскладка еще не начата, это комментарий в заголовке
				if inHeader {
					layouts.FileHeaderComments = append(layouts.FileHeaderComments, originalLine)
				} else {
					// Комментарии до раскладки сохраняем как PreComments для следующей раскладки
					preLayoutComments = append(preLayoutComments, originalLine)
				}
			} else if rowCount == 3 {
				// Комментарии после полной раскладки сохраняем как PostComments
				currentLayout.PostComments = append(currentLayout.PostComments, originalLine)
			} else {
				// Комментарии между строками раскладки или до завершения раскладки
				// сохраняем как PreComments
				preLayoutComments = append(preLayoutComments, originalLine)
			}
			continue
		}

		// Пропускаем пустые строки
		if trimmedLine == "" {
			if currentLayout != nil && rowCount == 3 {
				// Завершаем текущую раскладку
				currentLayout.PreComments = preLayoutComments
				layouts.Layouts = append(layouts.Layouts, *currentLayout)
				currentLayout = nil
				rowCount = 0
				preLayoutComments = []string{}  // Сбрасываем комментарии перед следующей раскладкой
				inHeader = false  // Больше не в заголовке
			}
			continue
		}

		// Если это название раскладки (первая строка после пустой или начало файла)
		if currentLayout == nil {
			inHeader = false  // Больше не в заголовке
			currentLayout = &Layout{
				Name: line,  // Сохраняем оригинальную строку с возможными комментариями в конце
			}
			rowCount = 0
			// Применяем накопленные комментарии перед раскладкой
			currentLayout.PreComments = preLayoutComments
			preLayoutComments = []string{}  // Сбрасываем, чтобы не использовать повторно
		} else if rowCount < 3 {
			// Парсим строку раскладки (включая возможные комментарии в конце строки)
			currentLayout.Keys[rowCount] = parseLayoutRowWithComments(line)
			rowCount++
		}
	}

	// Добавляем последнюю раскладку, если она есть
	if currentLayout != nil && rowCount == 3 {
		currentLayout.PreComments = preLayoutComments
		layouts.Layouts = append(layouts.Layouts, *currentLayout)
	}

	if len(layouts.Layouts) == 0 {
		return nil, fmt.Errorf("не найдено ни одной раскладки")
	}

	return layouts, nil
}

// WriteLayoutsToFile записывает раскладки в файл с сохранением комментариев
func WriteLayoutsToFile(parsedLayouts *ParsedLayouts, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("ошибка создания файла %s: %v", filename, err)
	}
	defer file.Close()

	// Записываем комментарии в начале файла
	for _, comment := range parsedLayouts.FileHeaderComments {
		if _, err := file.WriteString(comment + "\n"); err != nil {
			return fmt.Errorf("ошибка записи заголовочного комментария: %v", err)
		}
	}

	// Если есть заголовочные комментарии, добавляем пустую строку перед первой раскладкой
	if len(parsedLayouts.FileHeaderComments) > 0 && len(parsedLayouts.Layouts) > 0 {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("ошибка записи разделителя: %v", err)
		}
	}

	// Записываем раскладки
	for i, layout := range parsedLayouts.Layouts {
		// Записываем комментарии перед раскладкой
		for _, comment := range layout.PreComments {
			if _, err := file.WriteString(comment + "\n"); err != nil {
				return fmt.Errorf("ошибка записи комментария: %v", err)
			}
		}

		// Записываем имя раскладки
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

		// Записываем комментарии после раскладки
		for _, comment := range layout.PostComments {
			if _, err := file.WriteString(comment + "\n"); err != nil {
				return fmt.Errorf("ошибка записи комментария: %v", err)
			}
		}

		// Добавляем пустую строку-разделитель между раскладками, кроме последней
		if i < len(parsedLayouts.Layouts)-1 {
			if _, err := file.WriteString("\n"); err != nil {
				return fmt.Errorf("ошибка записи разделителя: %v", err)
			}
		}
	}

	return nil
}

// parseLayoutRowWithComments разбирает строку раскладки, извлекая клавиши и комментарии
func parseLayoutRowWithComments(line string) [10]string {
	// Ищем комментарий в конце строки (после #)
	commentStart := strings.Index(line, "#")
	var layoutPart string
	if commentStart != -1 {
		layoutPart = strings.TrimSpace(line[:commentStart])
	} else {
		layoutPart = strings.TrimSpace(line)
	}

	// Разбираем клавиши
	parts := strings.Fields(layoutPart)
	var result [10]string

	for i := 0; i < 10; i++ {
		if i < len(parts) {
			result[i] = parts[i]
		} else {
			result[i] = ""  // Пустая клавиша, если недостаточно элементов
		}
	}

	return result
}

// FindAndReplaceLayout находит раскладку по имени и заменяет её на новую
func FindAndReplaceLayout(parsedLayouts *ParsedLayouts, newLayout Layout) bool {
	for i := range parsedLayouts.Layouts {
		if parsedLayouts.Layouts[i].Name == newLayout.Name {
			// Сохраняем оригинальные комментарии
			originalPreComments := parsedLayouts.Layouts[i].PreComments
			originalPostComments := parsedLayouts.Layouts[i].PostComments
			// Заменяем раскладку, но сохраняем комментарии
			parsedLayouts.Layouts[i] = newLayout
			parsedLayouts.Layouts[i].PreComments = originalPreComments
			parsedLayouts.Layouts[i].PostComments = originalPostComments
			return true
		}
	}
	return false
}

// FindAndReplaceLayoutByIndex находит раскладку по индексу и заменяет её на новую
func FindAndReplaceLayoutByIndex(parsedLayouts *ParsedLayouts, index int, newLayout Layout) bool {
	if index < 0 || index >= len(parsedLayouts.Layouts) {
		return false
	}

	// Сохраняем оригинальные комментарии
	originalPreComments := parsedLayouts.Layouts[index].PreComments
	originalPostComments := parsedLayouts.Layouts[index].PostComments
	// Заменяем раскладку, но сохраняем комментарии
	parsedLayouts.Layouts[index] = newLayout
	parsedLayouts.Layouts[index].PreComments = originalPreComments
	parsedLayouts.Layouts[index].PostComments = originalPostComments
	return true
}

// LoadEffortMatrix загружает только матрицу усилий из файла
func LoadEffortMatrix(filename string) ([3][10]float64, error) {
	var effortMatrix [3][10]float64

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return effortMatrix, fmt.Errorf("ошибка при чтения файла матрицы усилий: %w", err)
	}

	// Разбиваем файл на строки
	allLines := strings.Split(strings.TrimSpace(string(file)), "\n")

	// Фильтруем строки, исключая комментарии и пустые строки
	var lines []string
	for _, line := range allLines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
			lines = append(lines, line)
		}
	}

	for row := 0; row < 3; row++ {
		if row >= len(lines) {
			return effortMatrix, fmt.Errorf("недостаточно строк для матрицы усилий")
		}

		parts := strings.Fields(lines[row])
		if len(parts) < 10 {
			return effortMatrix, fmt.Errorf("строка %d имеет менее 10 значений усилий", row+1)
		}

		for col := 0; col < 10; col++ {
			val, err := strconv.ParseFloat(parts[col], 64)
			if err != nil {
				return effortMatrix, fmt.Errorf("ошибка парсинга усилия [%d][%d]: %w", row, col, err)
			}
			effortMatrix[row][col] = val
		}
	}

	return effortMatrix, nil
}

// LoadAllData загружает все необходимые данные
func LoadAllData(langFile, configFile, layoutFile string) (*LanguageData, *KeyboardConfig, *ParsedLayouts, error) {
	langData, err := LoadLanguageData(langFile)
	if err != nil {
		return nil, nil, nil, err
	}

	config, err := LoadKeyboardConfig(configFile)
	if err != nil {
		return nil, nil, nil, err
	}

	layouts, err := LoadLayouts(layoutFile)
	if err != nil {
		return nil, nil, nil, err
	}

	return langData, config, layouts, nil
}
