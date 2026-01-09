package main

// LanguageData содержит данные о языке - частоты букв и биграмм
type LanguageData struct {
	Language   string             `json:"language"`
	Characters map[string]float64 `json:"characters"`
	Bigrams    map[string]float64 `json:"bigrams"`
}

// KeyboardConfig содержит конфигурацию клавиатуры
type KeyboardConfig struct {
	EffortMatrix           [3][10]float64 // Матрица усилий (3 ряда x 10 столбцов)
	FixedPositions         [3][10]string  // Матрица фиксированных позиций ('x' или '.')
	MaxFingerEfforts       [8]float64     // Максимальное значение усилия для каждого пальца
	FingerEffortPenalties  [8]float64     // Значения штрафа за превышение максимальной нагрузки для каждого пальца
	MaxRowEfforts          [3]float64     // Максимальные значения усилия для каждого ряда (MR1, MR2, MR3)
	RowEffortPenalties     [3]float64     // Значения штрафа за превышение максимальной нагрузки для каждого ряда (PR1, PR2, PR3)
	Weights                WeightConfig   // Коэффициенты весов для параметров
	BigramIndividualCoeffs []BigramIndividualCoeff // Индивидуальные коэффициенты для отдельных биграмм
}

// BigramIndividualCoeff структура для хранения индивидуального коэффициента для биграммы
type BigramIndividualCoeff struct {
	Pos1 int     // Номер позиции первого символа (0-29)
	Pos2 int     // Номер позиции второго символа (0-29)
	Coeff float64 // Индивидуальный коэффициент
}

// WeightConfig содержит веса для расчёта оценки раскладки
type WeightConfig struct {
	Effort         float64
	HandSwitch     float64
	SameFinger     float64
	SameFingerJump float64
	Inroll         float64
	Outroll        float64
	// Нормирующие коэффициенты для биграмм
	SHB             float64 // Same Hand Bigram
	SFB             float64 // Same Finger Bigrams
	HVB             float64 // Half Vertical Bigrams
	FVB             float64 // Full Vertical Bigrams
	HDB             float64 // Half Diagonal Bigrams
	FDB             float64 // Full Diagonal Bigrams
	HFB             float64 // Horizontal Finger Bigrams
	HSB             float64 // Half Scissors Bigrams
	FSB             float64 // Full Scissors Bigrams
	LSB             float64 // Lateral Stretch Bigram
	SRB             float64 // Same Row Bigrams
	AFI             float64 // Adjacent Fingers In (соседние клавиши в одном ряду нажимаются по направлению к центру)
	AFO             float64 // Adjacent Fingers Out (соседние клавиши в одном ряду нажимаются по направлению от центра)
	HDI             float64 // Hand Disbalance Index
	FDI             float64 // Finger Disbalance Index
	D18             float64 // Coefficient for disbalance between fingers 1 and 8
	D27             float64 // Coefficient for disbalance between fingers 2 and 7
	D36             float64 // Coefficient for disbalance between fingers 3 and 6
	D45             float64 // Coefficient for disbalance between fingers 4 and 5
	HSBStrictMode   int     // Strict mode for HSB calculation (1=strict, 0=non-strict)
	FSBStrictMode   int     // Strict mode for FSB calculation (1=strict, 0=non-strict)
	LSBStrictMode   int     // Strict mode for LSB calculation (1=strict, 0=non-strict)
	TotalEffortNorm float64 // Нормализующий коэффициент для общего усилия
	// Дополнительные параметры для MEP
	MaxRowEffort1   float64 // Максимальное усилие для 1 ряда (MR1)
	MaxRowEffort2   float64 // Максимальное усилие для 2 ряда (MR2)
	MaxRowEffort3   float64 // Максимальное усилие для 3 ряда (MR3)
	RowPenalty1     float64 // Штраф для 1 ряда за превышение максимального усилия (PR1)
	RowPenalty2     float64 // Штраф для 2 ряда за превышение максимального усилия (PR2)
	RowPenalty3     float64 // Штраф для 3 ряда за превышение максимального усилия (PR3)
}

// Layout представляет одну раскладку
type Layout struct {
	Name        string
	Keys        [3][10]string // 3 ряда x 10 столбцов
	PreComments []string      // Комментарии перед раскладкой
	PostComments []string      // Комментарии после раскладки
}

// LayoutAnalysis содержит результаты анализа раскладки
type LayoutAnalysis struct {
	LayoutName     string
	LayoutIndex    int        // Индекс раскладки в файле (1-based)
	TotalEffort    float64    // Суммарное усилие (%)
	EffortByRow    [3]float64 // Усилие по рядам (%)
	EffortByFinger [8]float64 // Усилие по пальцам (%)
	EffortByHalf   [2]float64 // Усилие по половинкам (%)
	BigramAnalysis BigramAnalysis
	HDI            float64       // Hand Disbalance Index
	FDI            float64       // Finger Disbalance Index
	MEP            float64       // Maximum Effort Penalty
	WeightedScore  float64       // Итоговая взвешенная оценка
	Config         *KeyboardConfig // Reference to the configuration for accessing weights
}

// BigramAnalysis содержит анализ биграмм
type BigramAnalysis struct {
	SHB  float64 // Same Hand Bigram
	SFB  float64 // Same Finger Bigrams
	HVB  float64 // Half Vertical Bigrams
	FVB  float64 // Full Vertical Bigrams
	HDB  float64 // Half Diagonal Bigrams
	FDB  float64 // Full Diagonal Bigrams
	HFB  float64 // Horizontal Finger Bigrams
	HSB  float64 // Half Scissors Bigrams
	FSB  float64 // Full Scissors Bigrams
	LSB  float64 // Lateral Stretch Bigram
	SRB  float64 // Same Row Bigrams
	AFI  float64 // Adjacent Fingers In (соседние клавиши в одном ряду нажимаются по направлению к центру)
	AFO  float64 // Adjacent Fingers Out (соседние клавиши в одном ряду нажимаются по направлению от центра)
	// Дополнительные параметры для нестрогого режима
	HSB2 float64 // Half Scissors Bigrams (вне строгого режима)
	FSB2 float64 // Full Scissors Bigrams (вне строгого режима)
	LSB2 float64 // Lateral Stretch Bigrams (вне строгого режима)
	SKB  float64 // Same Key Bigrams
	TIB  float64 // Total on Individual Bigrams (сумма частот биграмм с индивидуальными коэффициентами, домноженных на соответствующий коэффициент)
}

// ParsedLayouts содержит все загруженные раскладки
type ParsedLayouts struct {
	Layouts []Layout
	FileHeaderComments []string  // Комментарии в начале файла до первой раскладки
}

// BigramFreq структура для хранения биграммы и её частоты
type BigramFreq struct {
	Bigram string
	Freq   float64
}

// Equals сравнивает два объекта Layout
func (l *Layout) Equals(other *Layout) bool {
	if l.Name != other.Name {
		return false
	}

	for row := 0; row < 3; row++ {
		for col := 0; col < 10; col++ {
			if l.Keys[row][col] != other.Keys[row][col] {
				return false
			}
		}
	}

	return true
}
