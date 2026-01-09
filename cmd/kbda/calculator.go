package main

import (
	"fmt"
	"math"
	"regexp"
)

// Finger indices for finger assignment
const (
	FingerThumb1  = 0 // Left thumb
	FingerIndex1  = 1 // Left index
	FingerMiddle1 = 2 // Left middle
	FingerRing1   = 3 // Left ring
	FingerPinky1  = 4 // Left pinky
	FingerThumb2  = 5 // Right thumb
	FingerIndex2  = 6 // Right index
	FingerMiddle2 = 7 // Right middle
	FingerRing2   = 8 // Right ring
	FingerPinky2  = 9 // Right pinky
)

// getFingerForKey возвращает палец для позиции (row, col)
// Разметка пальцев: 1 2 3 4 4  5 5 6 7 8
// Индексы: 0 1 2 3 4  5 6 7 8 - но нужны P1-P8 (0-7)
// Левая: P1(0) P2(1) P3(2) P4(3) P4(3)  P5(4) P5(4) P6(5) P7(6) P8(7)
// Правая в исходной нотации - но мы используем единую нумерацию 0-7
func getFingerForKey(row, col int) int {
	// Палец для каждой колонки (левая рука: 0-4, правая: 5-9)
	// Левая: P1, P2, P3, P4, P4  Правая: P5, P5, P6, P7, P8
	// В индексах 0-7: 0, 1, 2, 3, 3  4, 4, 5, 6, 7
	fingerMap := []int{0, 1, 2, 3, 3, 4, 4, 5, 6, 7}
	return fingerMap[col]
}

// getHalf возвращает номер половинки (0 - левая, 1 - правая)
func getHalf(col int) int {
	if col < 5 {
		return 0
	}
	return 1
}

// AnalyzeLayout анализирует раскладку и возвращает результаты
func AnalyzeLayout(layout *Layout, config *KeyboardConfig, langData *LanguageData) *LayoutAnalysis {
	analysis := &LayoutAnalysis{
		LayoutName: layout.Name,
		Config:     config,
	}

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

	// Рассчитываем суммарное усилие
	calculateEffort(layout, config, langData, keyPos, analysis)

	// Рассчитываем биграммы
	calculateBigrams(layout, config, langData, keyPos, analysis)

	// Рассчитываем HDI как разницу между нагрузкой на левую и правую руки
	analysis.HDI = math.Abs(analysis.EffortByHalf[0] - analysis.EffortByHalf[1])

	// Рассчитываем FDI как сумму разниц нагрузки по каждой паре пальцев на разных руках
	analysis.FDI = calculateFDI(analysis, config)

	// Рассчитываем MEP (Maximum Effort Penalty) как сумму превышений нагрузки по всем пальцам, домноженных на величину штрафа для каждого пальца
	analysis.MEP = calculateMEP(analysis, config)

	// Рассчитываем взвешенную оценку
	calculateWeightedScore(config, analysis)

	return analysis
}

// calculateEffort рассчитывает усилие для раскладки
func calculateEffort(layout *Layout, config *KeyboardConfig, langData *LanguageData, keyPos map[string][2]int, analysis *LayoutAnalysis) {
	totalEffort := 0.0
	totalFreq := 0.0

	// Рассчитываем суммарное усилие, нормированное на единицу
	for char, freq := range langData.Characters {
		pos, exists := keyPos[char]
		if !exists {
			continue // Пропускаем символы, не присутствующие в раскладке
		}

		row, col := pos[0], pos[1]
		effort := config.EffortMatrix[row][col]
		totalEffort += effort * freq
		totalFreq += freq
	}

	// Усилие для равномерного распределения (каждая буква 1/30)
	uniformEffort := 0.0
	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			uniformEffort += config.EffortMatrix[row][col]
		}
	}
	uniformEffort /= 30.0

	if totalFreq > 0 {
		// Нормируем на усилие равномерного распределения
		// Результат в процентах от 0.95 (если усилие равномерное)
		avgEffort := totalEffort / totalFreq
		analysis.TotalEffort = avgEffort / uniformEffort * 100.0
	}

	// Рассчитываем усилия по рядам так, чтобы они суммировались в 100%
	rowTotalEfforts := [3]float64{}
	rowTotalFreqs := [3]float64{}
	for char, freq := range langData.Characters {
		pos, exists := keyPos[char]
		if !exists {
			continue
		}
		row := pos[0]
		rowTotalEfforts[row] += config.EffortMatrix[pos[0]][pos[1]] * freq
		rowTotalFreqs[row] += freq
	}

	if totalFreq > 0 {
		for row := 0; row < 3; row++ {
			analysis.EffortByRow[row] = (rowTotalFreqs[row] / totalFreq) * 100.0
		}
	}

	// Рассчитываем усилия по пальцам так, чтобы они суммировались в 100%
	// Пальцы: 0=P1, 1=P2, 2=P3, 3=P4, 4=P5, 5=P6, 6=P7, 7=P8
	fingerFreqs := [8]float64{}
	for char, freq := range langData.Characters {
		pos, exists := keyPos[char]
		if !exists {
			continue
		}
		row, col := pos[0], pos[1]
		f := getFingerForKey(row, col)
		if f < 8 {
			fingerFreqs[f] += freq
		}
	}

	if totalFreq > 0 {
		for finger := 0; finger < 8; finger++ {
			analysis.EffortByFinger[finger] = (fingerFreqs[finger] / totalFreq) * 100.0
		}
	}

	// Усилие по половинкам так, чтобы суммировались в 100%
	halfFreqs := [2]float64{}
	for char, freq := range langData.Characters {
		pos, exists := keyPos[char]
		if !exists {
			continue
		}
		h := getHalf(pos[1])
		halfFreqs[h] += freq
	}

	if totalFreq > 0 {
		for half := 0; half < 2; half++ {
			analysis.EffortByHalf[half] = (halfFreqs[half] / totalFreq) * 100.0
		}
	}
}

// calculateBigrams рассчитывает анализ биграмм
func calculateBigrams(layout *Layout, config *KeyboardConfig, langData *LanguageData, keyPos map[string][2]int, analysis *LayoutAnalysis) {

	// Новые метрики биграмм
	shb := 0.0  // Same Hand Bigram (процент биграмм, которые набираются одной рукой)
	sfb := 0.0  // Same Finger Bigrams
	hvb := 0.0  // Half Vertical Bigrams (один палец, одна колонка, соседние ряды, исключая колонки 5 и 6)
	fvb := 0.0  // Full Vertical Bigrams (один палец, одна колонка, через ряд, исключая колонки 5 и 6)
	hdb := 0.0  // Half Diagonal Bigrams (один палец, соседние колонки и соседние ряды)
	fdb := 0.0  // Full Diagonal Bigrams (один палец, соседние колонки через ряд)
	hfb := 0.0  // Horizontal Finger Bigrams (один палец, один ряд, соседние колонки)
	hsb := 0.0  // Half Scissors Bigrams (одна рука, разные пальцы, соседние ряды, один из пальцев 2,3,6,7, исключая колонки 5 и 6)
	fsb := 0.0  // Full Scissors Bigrams (одна рука, разные пальцы, 1 и 3 ряд, один из пальцев 2,3,6,7, исключая колонки 5 и 6)
	lsb := 0.0  // Lateral Stretch Bigram (указательный и средний на одной руке через вертикальный ряд, колонки 3-5 или 6-8)
	srb := 0.0  // Same Row Bigrams (одна рука, один ряд, исключая колонки 5 и 6)
	afi := 0.0  // Adjacent Fingers In (соседние клавиши в одном ряду нажимаются по направлению к центру)
	afo := 0.0  // Adjacent Fingers Out (соседние клавиши в одном ряду нажимаются по направлению от центра)

	// Дополнительные параметры для нестрогого режима
	hsb2 := 0.0  // Half Scissors Bigrams (вне строгого режима)
	fsb2 := 0.0  // Full Scissors Bigrams (вне строгого режима)
	lsb2 := 0.0  // Lateral Stretch Bigrams (вне строгого режима)
	skb := 0.0   // Same Key Bigrams

	totalBigramFreq := 0.0

	// Проход по всем биграммам
	for bigram, freq := range langData.Bigrams {
		runes := []rune(bigram)
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

		totalBigramFreq += freq

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
			shb += freq
		}

		// SFB - Same Finger Bigrams (процент биграмм, которые набираются одним пальцем)
		if finger1 == finger2 {
			sfb += freq
		}

		// Рассчитываем метрики только если оба символа на одной половинке
		if half1 == half2 {
			// HVB - Half Vertical Bigrams (один палец, одна колонка, соседние ряды, исключая колонки 5 и 6)
			if finger1 == finger2 && col1 == col2 && rowDiff == 1 && col1 != 4 && col1 != 5 {
				hvb += freq
			}

			// FVB - Full Vertical Bigrams (один палец, одна колонка, через ряд, исключая колонки 5 и 6)
			if finger1 == finger2 && col1 == col2 && rowDiff == 2 && col1 != 4 && col1 != 5 {
				fvb += freq
			}

			// HDB - Half Diagonal Bigrams (один палец, соседние колонки и соседние ряды)
			if finger1 == finger2 && rowDiff == 1 && colDiff == 1 {
				hdb += freq
			}

			// FDB - Full Diagonal Bigrams (один палец, соседние колонки через ряд)
			if finger1 == finger2 && rowDiff == 2 && colDiff == 1 {
				fdb += freq
			}

			// HFB - Horizontal Finger Bigrams (один палец, один ряд, соседние колонки)
			if finger1 == finger2 && row1 == row2 && colDiff == 1 {
				hfb += freq
			}

			// SRB - Same Row Bigrams (одна рука, один ряд, исключая колонки 5 и 6)
			if row1 == row2 && col1 != 4 && col1 != 5 && col2 != 4 && col2 != 5 {
				srb += freq
			}

			// Определяем специфичные пальцы (2, 3, 6, 7 - индексы 1, 2, 5, 6)
			finger1IsSpecial := (finger1 == 1 || finger1 == 2 || finger1 == 5 || finger1 == 6)
			finger2IsSpecial := (finger2 == 1 || finger2 == 2 || finger2 == 5 || finger2 == 6)

			// HSB - Half Scissors Bigrams (одна рука, разные пальцы, соседние ряды, один из пальцев 2,3,6,7, исключая колонки 5 и 6)
			// Проверяем, что обе клавиши находятся на одной руке (левой: колонки 0-3 или правой: колонки 6-9)
			leftHandCols1 := col1 >= 0 && col1 <= 3
			rightHandCols1 := col1 >= 6 && col1 <= 9
			leftHandCols2 := col2 >= 0 && col2 <= 3
			rightHandCols2 := col2 >= 6 && col2 <= 9

			isSameHand := (leftHandCols1 && leftHandCols2) || (rightHandCols1 && rightHandCols2)

			if isSameHand && finger1 != finger2 && rowDiff == 1 && col1 != 4 && col1 != 5 && col2 != 4 && col2 != 5 {
				// Проверяем, находится ли нижний из двух рядов на специфичном пальце (2,3,6,7)
				lowerRow := row1
				if row2 > row1 {
					lowerRow = row2
				}  // Нижний (с большим номером) из двух рядов

				isHSBValid := (lowerRow == row1 && finger1IsSpecial) || (lowerRow == row2 && finger2IsSpecial)

				if config.Weights.HSBStrictMode == 1 {
					// Строгий режим: только если соответствует критериям
					if isHSBValid {
						hsb += freq
					} else {
						// Если не соответствует строгому режиму, добавляем к HSB2
						hsb2 += freq
					}
				} else {
					// Нестрогий режим: всегда добавляем к основному HSB, если это HSB паттерн
					hsb += freq
					// HSB2 остается равным 0 в нестрогом режиме
				}
			}

			// FSB - Full Scissors Bigrams (одна рука, разные пальцы, 1 и 3 ряд, один из пальцев 2, 3, 6 или 7, исключая колонки 5 и 6)
			// Проверяем, что обе клавиши находятся на одной руке (левой: колонки 0-3 или правой: колонки 6-9)
			if isSameHand && finger1 != finger2 && rowDiff == 2 && ((row1 == 0 && row2 == 2) || (row1 == 2 && row2 == 0)) && col1 != 4 && col1 != 5 && col2 != 4 && col2 != 5 {
				// Проверяем, находится ли 3-й ряд (индекс 2) на специфичном пальце (2,3,6,7)
				isFSBValid := (row1 == 2 && finger1IsSpecial) || (row2 == 2 && finger2IsSpecial)

				if config.Weights.FSBStrictMode == 1 {
					// Строгий режим: только если соответствует критериям
					if isFSBValid {
						fsb += freq
					} else {
						// Если не соответствует строгому режиму, добавляем к FSB2
						fsb2 += freq
					}
				} else {
					// Нестрогий режим: всегда добавляем к основному FSB, если это FSB паттерн
					fsb += freq
					// FSB2 остается равным 0 в нестрогом режиме
				}
			}

			// LSB - Lateral Stretch Bigram (указательный и средний на одной руке через вертикальный ряд, колонки 3-5 или 6-8)
			// Палец 1 = колонка 1 (индекс 1), Палец 2 = колонка 2 (индекс 2), Палец 5 = колонка 7 (индекс 6), Палец 6 = колонка 8 (индекс 7)
			// Это колонки 2-4 или 5-7 (в индексах 0-9)
			isLSBPattern := (col1 == 2 && col2 == 4) || (col1 == 4 && col2 == 2) || (col1 == 5 && col2 == 7) || (col1 == 7 && col2 == 5) // колонки 3-5 или 6-8

			if isLSBPattern {
				// Проверяем, что один символ находится на указательном пальце (2 или 5), а другой на среднем (3 или 6)
				finger1IsIndex := (finger1 == 1 || finger1 == 4) // Палец 2 или 5 (указательный)
				finger2IsMiddle := (finger2 == 2 || finger2 == 5) // Палец 3 или 6 (средний)

				finger2IsIndex := (finger2 == 1 || finger2 == 4) // Палец 2 или 5 (указательный)
				finger1IsMiddle := (finger1 == 2 || finger1 == 5) // Палец 3 или 6 (средний)

				isLSBValid := (finger1IsIndex && finger2IsMiddle) || (finger2IsIndex && finger1IsMiddle)

				if config.Weights.LSBStrictMode == 1 {
					// Строгий режим: только если соответствует критериям
					if isLSBValid {
						lsb += freq
					} else {
						// Если не соответствует строгому режиму, добавляем к LSB2
						lsb2 += freq
					}
				} else {
					// Нестрогий режим: всегда добавляем к основному LSB, если это LSB паттерн
					lsb += freq
					// LSB2 остается равным 0 в нестрогом режиме
				}
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
					afi += freq // движение к центру
				} else if centerDist1 < centerDist2 { // первый символ ближе к центру
					afo += freq // движение от центра
				}
			}
		}

		// SKB - Same Key Bigrams (биграммы с одинаковыми символами)
		if char1 == char2 {
			skb += freq
		}

		// Проверяем, есть ли для этой биграммы индивидуальный коэффициент
		// Для этого нужно определить позиции символов в раскладке
		if pos1, exists1 := keyPos[char1]; exists1 {
			if pos2, exists2 := keyPos[char2]; exists2 {
				// Преобразуем позиции в номера (1-30)
				pos1Num := pos1[0]*10 + pos1[1] + 1  // row*10 + col + 1
				pos2Num := pos2[0]*10 + pos2[1] + 1  // row*10 + col + 1

				// Ищем индивидуальный коэффициент для этой биграммы
				for _, coeff := range config.BigramIndividualCoeffs {
					// Проверяем, совпадает ли биграмма с заданной (с учетом порядка)
					if coeff.Pos1 == pos1Num-1 && coeff.Pos2 == pos2Num-1 {  // Преобразуем обратно к индексам (0-29)
						analysis.BigramAnalysis.TIB += freq * coeff.Coeff
					}
				}
			}
		}
	}

	if totalBigramFreq > 0 {
		// Нормируем все значения на общую частоту биграмм
		analysis.BigramAnalysis.SHB = (shb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.SFB = (sfb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.HVB = (hvb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.FVB = (fvb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.HDB = (hdb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.FDB = (fdb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.HFB = (hfb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.HSB = (hsb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.FSB = (fsb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.LSB = (lsb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.SRB = (srb / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.AFI = (afi / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.AFO = (afo / totalBigramFreq) * 100.0
		// Дополнительные параметры
		analysis.BigramAnalysis.HSB2 = (hsb2 / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.FSB2 = (fsb2 / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.LSB2 = (lsb2 / totalBigramFreq) * 100.0
		analysis.BigramAnalysis.SKB = (skb / totalBigramFreq) * 100.0
		// TIB уже рассчитан в цикле по биграммам, нормируем его
		analysis.BigramAnalysis.TIB = (analysis.BigramAnalysis.TIB / totalBigramFreq) * 100.0
	}
}

// calculateBigramEffortSum calculates the sum of all bigram values multiplied by their corresponding coefficients
func calculateBigramEffortSum(config *KeyboardConfig, analysis *LayoutAnalysis) float64 {
	// Calculate sum of all bigram coefficients multiplied by their weights
	bigramEffort := 0.0
	bigramEffort += config.Weights.SHB * analysis.BigramAnalysis.SHB
	bigramEffort += config.Weights.SFB * analysis.BigramAnalysis.SFB
	bigramEffort += config.Weights.HVB * analysis.BigramAnalysis.HVB
	bigramEffort += config.Weights.FVB * analysis.BigramAnalysis.FVB
	bigramEffort += config.Weights.HDB * analysis.BigramAnalysis.HDB
	bigramEffort += config.Weights.FDB * analysis.BigramAnalysis.FDB
	bigramEffort += config.Weights.HFB * analysis.BigramAnalysis.HFB
	bigramEffort += config.Weights.HSB * analysis.BigramAnalysis.HSB
	bigramEffort += config.Weights.FSB * analysis.BigramAnalysis.FSB
	bigramEffort += config.Weights.LSB * analysis.BigramAnalysis.LSB
	bigramEffort += config.Weights.SRB * analysis.BigramAnalysis.SRB
	bigramEffort += config.Weights.AFI * analysis.BigramAnalysis.AFI
	bigramEffort += config.Weights.AFO * analysis.BigramAnalysis.AFO
	bigramEffort += analysis.BigramAnalysis.TIB  // Добавляем TIB к общей сумме

	return bigramEffort
}

// calculateWeightedScore рассчитывает взвешенную оценку
func calculateWeightedScore(config *KeyboardConfig, analysis *LayoutAnalysis) {
	score := 0.0

	// Вычисляем оценку как сумму общего усилия и всех коэффициентов для биграмм,
	// домноженных на соответствующие нормирующие коэффициенты
	score += config.Weights.TotalEffortNorm * analysis.TotalEffort

	// Добавляем коэффициенты биграмм
	score += config.Weights.SHB * analysis.BigramAnalysis.SHB
	score += config.Weights.SFB * analysis.BigramAnalysis.SFB
	score += config.Weights.HVB * analysis.BigramAnalysis.HVB
	score += config.Weights.FVB * analysis.BigramAnalysis.FVB
	score += config.Weights.HDB * analysis.BigramAnalysis.HDB
	score += config.Weights.FDB * analysis.BigramAnalysis.FDB
	score += config.Weights.HFB * analysis.BigramAnalysis.HFB
	score += config.Weights.HSB * analysis.BigramAnalysis.HSB
	score += config.Weights.FSB * analysis.BigramAnalysis.FSB
	score += config.Weights.LSB * analysis.BigramAnalysis.LSB
	score += config.Weights.SRB * analysis.BigramAnalysis.SRB
	score += config.Weights.AFI * analysis.BigramAnalysis.AFI
	score += config.Weights.AFO * analysis.BigramAnalysis.AFO
	score += analysis.BigramAnalysis.TIB  // Добавляем TIB к оценке
	score += config.Weights.HDI * analysis.HDI
	score += config.Weights.FDI * analysis.FDI
	score += analysis.MEP  // Добавляем штраф за превышение максимальной нагрузки

	analysis.WeightedScore = score
}

// calculateFDI рассчитывает Finger Disbalance Index
func calculateFDI(analysis *LayoutAnalysis, config *KeyboardConfig) float64 {
	// FDI рассчитывается как сумма разниц нагрузки по каждому пальцу на разных руках,
	// домноженных на соответствующий весовой коэффициент по каждой паре пальцев
	// Pairs: 1-8, 2-7, 3-6, 4-5
	fdi := 0.0

	// Пальцы: 0=P1, 1=P2, 2=P3, 3=P4, 4=P5, 5=P6, 6=P7, 7=P8
	// Left hand fingers: 0-4 (P1-P5), Right hand fingers: 5-7 (P6-P8)

	// Pairs:
	// P1 (0) - P8 (7)
	fdi += config.Weights.D18 * math.Abs(analysis.EffortByFinger[0] - analysis.EffortByFinger[7])
	// P2 (1) - P7 (6)
	fdi += config.Weights.D27 * math.Abs(analysis.EffortByFinger[1] - analysis.EffortByFinger[6])
	// P3 (2) - P6 (5)
	fdi += config.Weights.D36 * math.Abs(analysis.EffortByFinger[2] - analysis.EffortByFinger[5])
	// P4 (3) - P5 (4)
	fdi += config.Weights.D45 * math.Abs(analysis.EffortByFinger[3] - analysis.EffortByFinger[4])

	return fdi
}

// calculateMEP рассчитывает Maximum Effort Penalty
func calculateMEP(analysis *LayoutAnalysis, config *KeyboardConfig) float64 {
	// MEP рассчитывается как сумма превышений нагрузки по всем пальцам и рядам, домноженных на величину штрафа для каждого пальца/ряда
	mep := 0.0

	// Рассчитываем штрафы для пальцев
	for finger := 0; finger < 8; finger++ {
		maxFingerEffort := config.MaxFingerEfforts[finger]

		// Если максимальное усилие для пальца равно 0, штраф не применяется
		if maxFingerEffort > 0 {
			excess := analysis.EffortByFinger[finger] - maxFingerEffort
			if excess > 0 {
				mep += excess * config.FingerEffortPenalties[finger]
			}
		}
	}

	// Рассчитываем штрафы для рядов
	for row := 0; row < 3; row++ {
		maxRowEffort := config.MaxRowEfforts[row]

		// Если максимальное усилие для ряда равно 0, штраф не применяется
		if maxRowEffort > 0 {
			excess := analysis.EffortByRow[row] - maxRowEffort
			if excess > 0 {
				mep += excess * config.RowEffortPenalties[row]
			}
		}
	}

	return mep
}

// FormatAnalysis форматирует результаты анализа для вывода
func FormatAnalysis(analysis *LayoutAnalysis) string {
	// Выводим усилия по пальцам (8), рядам (3), половинкам (2), hdi, fdi, mep, общее усилие и score
	return fmt.Sprintf(
		"%-4s %-16s %5.1f %5.1f %5.1f %5.1f %5.1f %5.1f %5.1f %5.1f  %5.1f %5.1f %5.1f  %5.1f %5.1f  %4.1f %4.1f %5.1f %7.2f %7.2f",
		fmt.Sprintf("[%d]", analysis.LayoutIndex),
		analysis.LayoutName,
		analysis.EffortByFinger[0], analysis.EffortByFinger[1], analysis.EffortByFinger[2], analysis.EffortByFinger[3],
		analysis.EffortByFinger[4], analysis.EffortByFinger[5], analysis.EffortByFinger[6], analysis.EffortByFinger[7],
		analysis.EffortByRow[0], analysis.EffortByRow[1], analysis.EffortByRow[2],
		analysis.EffortByHalf[0], analysis.EffortByHalf[1], analysis.HDI, analysis.FDI, analysis.MEP,
		analysis.TotalEffort,   // Display as percentage without % sign
		analysis.WeightedScore, // Display as percentage without % sign
	)
}

// FormatBigramAnalysis форматирует результаты анализа биграмм для вывода в виде таблицы
func FormatBigramAnalysis(analysis *LayoutAnalysis) string {
	bigramEffortSum := calculateBigramEffortSum(analysis.Config, analysis)
	return fmt.Sprintf(
		"%-4s %-16s %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %8.2f %7.2f",
		fmt.Sprintf("[%d]", analysis.LayoutIndex),
		analysis.LayoutName,
		analysis.BigramAnalysis.SHB,  // SHB - Same Hand Bigram
		analysis.BigramAnalysis.SFB,  // SFB - Same Finger Bigrams
		analysis.BigramAnalysis.HVB,  // HVB - Half Vertical Bigrams
		analysis.BigramAnalysis.FVB,  // FVB - Full Vertical Bigrams
		analysis.BigramAnalysis.HDB,  // HDB - Half Diagonal Bigrams
		analysis.BigramAnalysis.FDB,  // FDB - Full Diagonal Bigrams
		analysis.BigramAnalysis.HFB,  // HFB - Horizontal Finger Bigrams
		analysis.BigramAnalysis.HSB,  // HSB - Half Scissors Bigrams
		analysis.BigramAnalysis.FSB,  // FSB - Full Scissors Bigrams
		analysis.BigramAnalysis.LSB,  // LSB - Lateral Stretch Bigram
		analysis.BigramAnalysis.SRB,  // SRB - Same Row Bigrams
		analysis.BigramAnalysis.AFI,  // AFI - Adjacent Fingers In
		analysis.BigramAnalysis.AFO,  // AFO - Adjacent Fingers Out
		analysis.BigramAnalysis.TIB,  // TIB - Total on Individual Bigrams
		bigramEffortSum,              // Sum of all bigram values multiplied by coefficients
		analysis.WeightedScore,       // Display as percentage
	)
}

// FormatAnalysisWithHighlights форматирует результаты анализа с подсветкой числовых значений
func FormatAnalysisWithHighlights(analysis *LayoutAnalysis) string {
	// Форматируем базовую строку
	baseString := fmt.Sprintf(
		"%-4s %-16s %5.1f %5.1f %5.1f %5.1f %5.1f %5.1f %5.1f %5.1f  %5.1f %5.1f %5.1f  %5.1f %5.1f  %4.1f %4.1f %5.1f %7.2f %7.2f",
		fmt.Sprintf("[%d]", analysis.LayoutIndex),
		analysis.LayoutName,
		analysis.EffortByFinger[0], analysis.EffortByFinger[1], analysis.EffortByFinger[2], analysis.EffortByFinger[3],
		analysis.EffortByFinger[4], analysis.EffortByFinger[5], analysis.EffortByFinger[6], analysis.EffortByFinger[7],
		analysis.EffortByRow[0], analysis.EffortByRow[1], analysis.EffortByRow[2],
		analysis.EffortByHalf[0], analysis.EffortByHalf[1], analysis.HDI, analysis.FDI, analysis.MEP,
		analysis.TotalEffort,   // Display as percentage without % sign
		analysis.WeightedScore, // Display as percentage without % sign
	)

	// Подсвечиваем числовые значения в строке зеленым цветом (158,206,88)
	// Ищем числовые значения и оборачиваем их в цветовой код
	return colorizeNumbers(baseString)
}

// FormatBigramAnalysisWithHighlights форматирует результаты анализа биграмм с подсветкой числовых значений
func FormatBigramAnalysisWithHighlights(analysis *LayoutAnalysis) string {
	bigramEffortSum := calculateBigramEffortSum(analysis.Config, analysis)

	// Форматируем базовую строку
	baseString := fmt.Sprintf(
		"%-4s %-16s %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %6.2f %8.2f %7.2f",
		fmt.Sprintf("[%d]", analysis.LayoutIndex),
		analysis.LayoutName,
		analysis.BigramAnalysis.SHB,  // SHB - Same Hand Bigram
		analysis.BigramAnalysis.SFB,  // SFB - Same Finger Bigrams
		analysis.BigramAnalysis.HVB,  // HVB - Half Vertical Bigrams
		analysis.BigramAnalysis.FVB,  // FVB - Full Vertical Bigrams
		analysis.BigramAnalysis.HDB,  // HDB - Half Diagonal Bigrams
		analysis.BigramAnalysis.FDB,  // FDB - Full Diagonal Bigrams
		analysis.BigramAnalysis.HFB,  // HFB - Horizontal Finger Bigrams
		analysis.BigramAnalysis.HSB,  // HSB - Half Scissors Bigrams
		analysis.BigramAnalysis.FSB,  // FSB - Full Scissors Bigrams
		analysis.BigramAnalysis.LSB,  // LSB - Lateral Stretch Bigram
		analysis.BigramAnalysis.SRB,  // SRB - Same Row Bigrams
		analysis.BigramAnalysis.AFI,  // AFI - Adjacent Fingers In
		analysis.BigramAnalysis.AFO,  // AFO - Adjacent Fingers Out
		analysis.BigramAnalysis.TIB,  // TIB - Total on Individual Bigrams
		bigramEffortSum,              // Sum of all bigram values multiplied by coefficients
		analysis.WeightedScore,       // Display as percentage
	)

	// Подсвечиваем числовые значения в строке зеленым цветом (158,206,88)
	return colorizeNumbers(baseString)
}

// colorizeNumbers подсвечивает числовые значения в строке зеленым цветом
func colorizeNumbers(str string) string {
	// Регулярное выражение для поиска чисел с плавающей точкой
	re := regexp.MustCompile(`\d+(\.\d+)?`)

	// Функция для замены чисел на цветные
	return re.ReplaceAllStringFunc(str, func(match string) string {
		return fmt.Sprintf("\033[38;2;158;206;88m%s\033[0m", match)
	})
}

// abs возвращает абсолютное значение
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// absFloat возвращает абсолютное значение для float64
func absFloat(x float64) float64 {
	return math.Abs(x)
}
