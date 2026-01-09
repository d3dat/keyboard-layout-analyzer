package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// SimulatedAnnealingParams содержит параметры алгоритма Simulated Annealing
type SimulatedAnnealingParams struct {
	InitialTemp   float64
	CoolingRate   float64
	Iterations    int
	Restarts      int
	RandomSeed    int64
}

// SimulatedAnnealingResult содержит результат поиска
type SimulatedAnnealingResult struct {
	Layout   Layout
	Analysis *LayoutAnalysis
	Score    float64
}

// DefaultSAParams возвращает параметры SA по умолчанию
func DefaultSAParams() SimulatedAnnealingParams {
	return SimulatedAnnealingParams{
		InitialTemp: 1000.0,
		CoolingRate: 0.995,
		Iterations:  10000,
		Restarts:    5,
		RandomSeed:  time.Now().UnixNano(),
	}
}

// SearchOptimalLayout выполняет поиск оптимальной раскладки
func SearchOptimalLayout(config *KeyboardConfig, langData *LanguageData, params SimulatedAnnealingParams, numBest int) []SimulatedAnnealingResult {
	rand.Seed(params.RandomSeed)

	// Создаём начальную случайную раскладку
	initialLayout := generateRandomLayout(config, langData)

	var bestResults []SimulatedAnnealingResult

	for restart := 0; restart < params.Restarts; restart++ {
		fmt.Printf("Рестарт %d/%d\n", restart+1, params.Restarts)

		currentLayout := initialLayout
		if restart > 0 {
			// Для остальных рестартов генерируем новую начальную раскладку
			currentLayout = generateRandomLayout(config, langData)
		}

		currentAnalysis := AnalyzeLayout(&currentLayout, config, langData)
		currentScore := currentAnalysis.WeightedScore

		bestLayoutRestart := currentLayout
		bestScoreRestart := currentScore
		bestAnalysisRestart := currentAnalysis

		temperature := params.InitialTemp

		for iter := 0; iter < params.Iterations; iter++ {
			// Генерируем соседнее решение
			neighborLayout := generateNeighbor(&currentLayout, config)
			neighborAnalysis := AnalyzeLayout(&neighborLayout, config, langData)
			neighborScore := neighborAnalysis.WeightedScore

			// Вычисляем дельту
			delta := neighborScore - currentScore

			// Принимаем или отклоняем соседнее решение
			if delta < 0 || rand.Float64() < math.Exp(-delta/temperature) {
				currentLayout = neighborLayout
				currentAnalysis = neighborAnalysis
				currentScore = neighborScore

				// Обновляем лучшее решение для этого рестарта
				if currentScore < bestScoreRestart {
					bestLayoutRestart = currentLayout
					bestScoreRestart = currentScore
					bestAnalysisRestart = currentAnalysis
				}
			}

			// Охлаждаем
			temperature *= params.CoolingRate

			if iter%1000 == 0 && iter > 0 {
				fmt.Printf("  Итерация %d/%d, Лучший score: %.2f, Текущий score: %.2f, Температура: %.2f\n",
					iter, params.Iterations, bestScoreRestart, currentScore, temperature)
			}
		}

		// Добавляем результат рестарта
		result := SimulatedAnnealingResult{
			Layout:   bestLayoutRestart,
			Analysis: bestAnalysisRestart,
			Score:    bestScoreRestart,
		}
		bestResults = append(bestResults, result)

		fmt.Printf("  Лучший score рестарта: %.2f\n", bestScoreRestart)
	}

	// Сортируем результаты по score (лучшие первыми)
	sortResultsByScore(bestResults)

	// Возвращаем только numBest результатов
	if numBest > 0 && len(bestResults) > numBest {
		bestResults = bestResults[:numBest]
	}

	return bestResults
}

// SearchOptimalLayoutFromLayouts выполняет поиск оптимальной раскладки, используя буквы из существующих раскладок
func SearchOptimalLayoutFromLayouts(config *KeyboardConfig, langData *LanguageData, layouts *ParsedLayouts, params SimulatedAnnealingParams, numBest int) []SimulatedAnnealingResult {
	rand.Seed(params.RandomSeed)

	// Создаём начальную случайную раскладку, используя буквы из существующих раскладок
	initialLayout := GenerateRandomLayoutFromLayouts(config, layouts, langData)

	var bestResults []SimulatedAnnealingResult

	for restart := 0; restart < params.Restarts; restart++ {
		fmt.Printf("Рестарт %d/%d\n", restart+1, params.Restarts)

		currentLayout := initialLayout
		if restart > 0 {
			// Для остальных рестартов генерируем новую начальную раскладку
			currentLayout = GenerateRandomLayoutFromLayouts(config, layouts, langData)
		}

		currentAnalysis := AnalyzeLayout(&currentLayout, config, langData)
		currentScore := currentAnalysis.WeightedScore

		bestLayoutRestart := currentLayout
		bestScoreRestart := currentScore
		bestAnalysisRestart := currentAnalysis

		temperature := params.InitialTemp

		for iter := 0; iter < params.Iterations; iter++ {
			// Генерируем соседнее решение
			neighborLayout := generateNeighbor(&currentLayout, config)
			neighborAnalysis := AnalyzeLayout(&neighborLayout, config, langData)
			neighborScore := neighborAnalysis.WeightedScore

			// Вычисляем дельту
			delta := neighborScore - currentScore

			// Принимаем или отклоняем соседнее решение
			if delta < 0 || rand.Float64() < math.Exp(-delta/temperature) {
				currentLayout = neighborLayout
				currentAnalysis = neighborAnalysis
				currentScore = neighborScore

				// Обновляем лучшее решение для этого рестарта
				if currentScore < bestScoreRestart {
					bestLayoutRestart = currentLayout
					bestScoreRestart = currentScore
					bestAnalysisRestart = currentAnalysis
				}
			}

			// Охлаждаем
			temperature *= params.CoolingRate

			if iter%1000 == 0 && iter > 0 {
				fmt.Printf("  Итерация %d/%d, Лучший score: %.2f, Текущий score: %.2f, Температура: %.2f\n",
					iter, params.Iterations, bestScoreRestart, currentScore, temperature)
			}
		}

		// Добавляем результат рестарта
		result := SimulatedAnnealingResult{
			Layout:   bestLayoutRestart,
			Analysis: bestAnalysisRestart,
			Score:    bestScoreRestart,
		}
		bestResults = append(bestResults, result)

		fmt.Printf("  Лучший score рестарта: %.2f\n", bestScoreRestart)
	}

	// Сортируем результаты по score (лучшие первыми)
	sortResultsByScore(bestResults)

	// Возвращаем только numBest результатов
	if numBest > 0 && len(bestResults) > numBest {
		bestResults = bestResults[:numBest]
	}

	return bestResults
}

// generateRandomLayout генерирует случайную раскладку
func generateRandomLayout(config *KeyboardConfig, langData *LanguageData) Layout {
	layout := Layout{
		Name: "random",
	}

	// Собираем все буквы, которые есть в языковых данных
	var letters []string
	for char := range langData.Characters {
		letters = append(letters, char)
	}

	// Перемешиваем буквы
	rand.Shuffle(len(letters), func(i, j int) {
		letters[i], letters[j] = letters[j], letters[i]
	})

	// Размещаем буквы в позициях, которые не зафиксированы
	letterIdx := 0
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if config.FixedPositions[row][col] == "." && letterIdx < len(letters) {
				layout.Keys[row][col] = letters[letterIdx]
				letterIdx++
			} else if config.FixedPositions[row][col] != "." && config.FixedPositions[row][col] != "x" {
				// Если это фиксированная буква
				layout.Keys[row][col] = config.FixedPositions[row][col]
			} else if config.FixedPositions[row][col] == "x" {
				// Если позиция зафиксирована как 'x', оставляем пустой
				// В этом случае, поскольку у нас нет исходных раскладок, позиция остается пустой
			}
		}
	}

	return layout
}

// GenerateRandomLayoutFromLayouts генерирует случайную раскладку, используя только буквы из существующих раскладок
func GenerateRandomLayoutFromLayouts(config *KeyboardConfig, layouts *ParsedLayouts, langData *LanguageData) Layout {
	layout := Layout{
		Name: "random",
	}

	// Собираем все уникальные буквы из первой раскладки
	lettersMap := make(map[string]bool)
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			key := layouts.Layouts[0].Keys[row][col]
			if key != "" && key != " " {
				lettersMap[key] = true
			}
		}
	}

	freeLetters := make([]string, 0)

	// Размещаем буквы в фиксированных 'x' позициях и собираем оставшиеся буквы
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if config.FixedPositions[row][col] == "x" && len(layouts.Layouts) > 0 {
				// Помещаем букву в фиксированную 'x' позицию
				layout.Keys[row][col] = layouts.Layouts[0].Keys[row][col]
				// Удаляем эту букву из пула свободных
				delete(lettersMap, layouts.Layouts[0].Keys[row][col])
			} else if config.FixedPositions[row][col] != "." && config.FixedPositions[row][col] != "x" {
				// Помещаем фиксированную букву (не 'x')
				layout.Keys[row][col] = config.FixedPositions[row][col]
				// Удаляем эту букву из пула свободных
				delete(lettersMap, config.FixedPositions[row][col])
			}
		}
	}

	// Собираем только свободные буквы
	for char := range lettersMap {
		freeLetters = append(freeLetters, char)
	}

	// Перемешиваем свободные буквы
	rand.Shuffle(len(freeLetters), func(i, j int) {
		freeLetters[i], freeLetters[j] = freeLetters[j], freeLetters[i]
	})

	// Размещаем свободные буквы в позициях, которые могут меняться (отмеченные '.')
	letterIdx := 0
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if config.FixedPositions[row][col] == "." && letterIdx < len(freeLetters) {
				layout.Keys[row][col] = freeLetters[letterIdx]
				letterIdx++
			}
		}
	}

	return layout
}

// generateNeighbor генерирует соседнее решение путём обмена двух букв
func generateNeighbor(layout *Layout, config *KeyboardConfig) Layout {
	neighbor := *layout

	// Находим две случайные позиции, которые не зафиксированы
	var swappablePositions [][2]int

	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if config.FixedPositions[row][col] == "." {
				swappablePositions = append(swappablePositions, [2]int{row, col})
			}
		}
	}

	if len(swappablePositions) < 2 {
		return neighbor
	}

	// Выбираем две случайные позиции
	idx1 := rand.Intn(len(swappablePositions))
	idx2 := rand.Intn(len(swappablePositions))

	// Если это одна и та же позиция, выбираем другую
	for idx2 == idx1 {
		idx2 = rand.Intn(len(swappablePositions))
	}

	pos1 := swappablePositions[idx1]
	pos2 := swappablePositions[idx2]

	// Обмениваем буквы
	neighbor.Keys[pos1[0]][pos1[1]], neighbor.Keys[pos2[0]][pos2[1]] =
		neighbor.Keys[pos2[0]][pos2[1]], neighbor.Keys[pos1[0]][pos1[1]]

	return neighbor
}

// sortResultsByScore сортирует результаты по score (лучшие первыми)
func sortResultsByScore(results []SimulatedAnnealingResult) {
	// Простая сортировка пузырьком
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score < results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// SearchOptimalLayoutFromSpecificLayout выполняет поиск оптимальной раскладки, используя заданную раскладку в качестве начальной точки
func SearchOptimalLayoutFromSpecificLayout(config *KeyboardConfig, langData *LanguageData, layouts *ParsedLayouts, params SimulatedAnnealingParams, numBest int, startLayout Layout) []SimulatedAnnealingResult {
	rand.Seed(params.RandomSeed)

	var bestResults []SimulatedAnnealingResult

	for restart := 0; restart < params.Restarts; restart++ {
		fmt.Printf("Рестарт %d/%d\n", restart+1, params.Restarts)

		// Use the specific starting layout
		currentLayout := startLayout
		if restart > 0 {
			// For additional restarts, we could use random layouts or the best layout from previous runs
			// For now, we'll continue using the same starting point or the best from previous iterations
			bestOfPrevious := currentLayout
			if len(bestResults) > 0 {
				bestOfPrevious = bestResults[0].Layout
			}
			currentLayout = bestOfPrevious
		}

		currentAnalysis := AnalyzeLayout(&currentLayout, config, langData)
		currentScore := currentAnalysis.WeightedScore

		bestLayoutRestart := currentLayout
		bestScoreRestart := currentScore
		bestAnalysisRestart := currentAnalysis

		temperature := params.InitialTemp

		for iter := 0; iter < params.Iterations; iter++ {
			// Generate neighboring solution - using only characters in the start layout
			neighborLayout := generateNeighborFromBaseLayout(&currentLayout, config, &startLayout)
			neighborAnalysis := AnalyzeLayout(&neighborLayout, config, langData)
			neighborScore := neighborAnalysis.WeightedScore

			// Calculate delta
			delta := neighborScore - currentScore

			// Accept or reject neighboring solution
			if delta < 0 || rand.Float64() < math.Exp(-delta/temperature) {
				currentLayout = neighborLayout
				currentAnalysis = neighborAnalysis
				currentScore = neighborScore

				// Update best solution for this restart
				if currentScore < bestScoreRestart {
					bestLayoutRestart = currentLayout
					bestScoreRestart = currentScore
					bestAnalysisRestart = currentAnalysis
				}
			}

			// Cooling
			temperature *= params.CoolingRate
		}

		// Add best result from this restart to results
		bestResults = append(bestResults, SimulatedAnnealingResult{
			Layout:   bestLayoutRestart,
			Score:    bestScoreRestart,
			Analysis: bestAnalysisRestart,
		})

		// Keep only best results
		sortResultsByScore(bestResults)
		if numBest > 0 && len(bestResults) > numBest {
			bestResults = bestResults[:numBest]
		}
	}

	// Return only numBest results
	if numBest > 0 && len(bestResults) > numBest {
		bestResults = bestResults[:numBest]
	}

	return bestResults
}



// SearchOptimalLayoutFromRandomLayout performs search for optimal layout starting from random layout ignoring fixed positions
func SearchOptimalLayoutFromRandomLayout(config *KeyboardConfig, langData *LanguageData, layouts *ParsedLayouts, params SimulatedAnnealingParams, numBest int) []SimulatedAnnealingResult {
	rand.Seed(params.RandomSeed)

	var bestResults []SimulatedAnnealingResult

	for restart := 0; restart < params.Restarts; restart++ {
		fmt.Printf("Рестарт %d/%d\n", restart+1, params.Restarts)

		// Create a random layout from only those characters present in existing layouts
		var letters []string

		// Extract characters from existing layouts
		charsMap := make(map[string]bool)
		for _, layout := range layouts.Layouts {
			for row := 0; row < 3; row++ {
				for col := 0; col < 10; col++ {
					key := layout.Keys[row][col]
					if key != "" && key != " " {
						charsMap[key] = true
					}
				}
			}
		}

		// Convert map to slice
		for char := range charsMap {
			letters = append(letters, char)
		}

		// Fallback to all available characters if no layouts exist
		if len(letters) == 0 {
			for char := range langData.Characters {
				letters = append(letters, char)
			}
		}

		// Shuffle the letters
		rand.Shuffle(len(letters), func(i, j int) {
			letters[i], letters[j] = letters[j], letters[i]
		})

		// Create layout and fill with random letters
		currentLayout := Layout{Name: fmt.Sprintf("random_%d", restart)}
		letterIdx := 0
		for row := 0; row < 3; row++ {
			for col := 0; col < 10; col++ {
				if letterIdx < len(letters) {
					currentLayout.Keys[row][col] = letters[letterIdx]
					letterIdx++
				} else {
					// Cycle back if we run out of letters
					currentLayout.Keys[row][col] = letters[letterIdx%len(letters)]
				}
			}
		}

		currentAnalysis := AnalyzeLayout(&currentLayout, config, langData)
		currentScore := currentAnalysis.WeightedScore

		bestLayoutRestart := currentLayout
		bestScoreRestart := currentScore
		bestAnalysisRestart := currentAnalysis

		temperature := params.InitialTemp

		for iter := 0; iter < params.Iterations; iter++ {
			// Generate neighboring solution - ignores fixed positions for random search
			neighborLayout := generateRandomNeighborIgnoreFixed(&currentLayout, config, langData)
			neighborAnalysis := AnalyzeLayout(&neighborLayout, config, langData)
			neighborScore := neighborAnalysis.WeightedScore

			// Calculate delta
			delta := neighborScore - currentScore

			// Accept or reject neighboring solution
			if delta < 0 || rand.Float64() < math.Exp(-delta/temperature) {
				currentLayout = neighborLayout
				currentAnalysis = neighborAnalysis
				currentScore = neighborScore

				// Update best solution for this restart
				if currentScore < bestScoreRestart {
					bestLayoutRestart = currentLayout
					bestScoreRestart = currentScore
					bestAnalysisRestart = currentAnalysis
				}
			}

			// Cooling
			temperature *= params.CoolingRate
		}

		// Add best result from this restart to results
		bestResults = append(bestResults, SimulatedAnnealingResult{
			Layout:   bestLayoutRestart,
			Score:    bestScoreRestart,
			Analysis: bestAnalysisRestart,
		})

		// Keep only best results
		sortResultsByScore(bestResults)
		if numBest > 0 && len(bestResults) > numBest {
			bestResults = bestResults[:numBest]
		}
	}

	// Return only numBest results
	if numBest > 0 && len(bestResults) > numBest {
		bestResults = bestResults[:numBest]
	}

	return bestResults
}

// generateNeighborFromBaseLayout generates a neighboring solution using only characters from the base layout
func generateNeighborFromBaseLayout(layout *Layout, config *KeyboardConfig, baseLayout *Layout) Layout {
	neighbor := *layout

	// Collect unique characters from the base layout
	uniqueChars := make(map[string]bool)
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			key := baseLayout.Keys[row][col]
			if key != "" && key != " " {
				uniqueChars[key] = true
			}
		}
	}

	// Get characters as a slice
	chars := make([]string, 0, len(uniqueChars))
	for char := range uniqueChars {
		chars = append(chars, char)
	}

	// Находим позиции, которые не зафиксированы и не пусты
	var swapPositions [][2]int
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if config.FixedPositions[row][col] == "." && layout.Keys[row][col] != "" && layout.Keys[row][col] != " " {
				swapPositions = append(swapPositions, [2]int{row, col})
			}
		}
	}

	if len(swapPositions) < 2 {
		return neighbor
	}

	// Выбираем две случайные позиции
	idx1 := rand.Intn(len(swapPositions))
	idx2 := rand.Intn(len(swapPositions))
	for idx2 == idx1 && len(swapPositions) > 1 {
		idx2 = rand.Intn(len(swapPositions))
	}

	pos1 := swapPositions[idx1]
	pos2 := swapPositions[idx2]

	// Обмениваем буквы
	neighbor.Keys[pos1[0]][pos1[1]], neighbor.Keys[pos2[0]][pos2[1]] =
		neighbor.Keys[pos2[0]][pos2[1]], neighbor.Keys[pos1[0]][pos1[1]]

	return neighbor
}

// generateRandomNeighborIgnoreFixed generates a neighboring solution without considering fixed positions
func generateRandomNeighborIgnoreFixed(layout *Layout, config *KeyboardConfig, langData *LanguageData) Layout {
	neighbor := *layout

	// Collect all available characters from language data
	allChars := make([]string, 0)
	for char := range langData.Characters {
		allChars = append(allChars, char)
	}

	if len(allChars) < 2 {
		return neighbor
	}

	// Find two random positions to swap (anywhere on the board, ignoring fixed positions)
	pos1Row := rand.Intn(3)
	pos1Col := rand.Intn(10)
	pos2Row := rand.Intn(3) 
	pos2Col := rand.Intn(10)

	// Ensure they are different positions
	for pos1Row == pos2Row && pos1Col == pos2Col {
		pos2Row = rand.Intn(3)
		pos2Col = rand.Intn(10)
	}

	// Swap the keys
	neighbor.Keys[pos1Row][pos1Col], neighbor.Keys[pos2Row][pos2Col] = 
		neighbor.Keys[pos2Row][pos2Col], neighbor.Keys[pos1Row][pos1Col]

	return neighbor
}
