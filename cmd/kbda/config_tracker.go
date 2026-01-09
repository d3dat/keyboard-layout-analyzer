package main

// ConfigChangeTracker отслеживает изменения в конфигурации
type ConfigChangeTracker struct {
    originalWeights WeightConfig        // Оригинальные веса из файла
    modifiedWeights WeightConfig        // Измененные веса
    weightsModified map[string]bool     // Какие веса были изменены
    allWeightNames  []string            // Все возможные имена весов
    originalBigramIndividualCoeffs []BigramIndividualCoeff // Оригинальные индивидуальные коэффициенты биграмм
    modifiedBigramIndividualCoeffs []BigramIndividualCoeff // Измененные индивидуальные коэффициенты биграмм
    bigramCoeffsModified bool            // Были ли изменены индивидуальные коэффициенты биграмм
}

// NewConfigChangeTracker создает новый трекер изменений
func NewConfigChangeTracker(weights WeightConfig) *ConfigChangeTracker {
    tracker := &ConfigChangeTracker{
        originalWeights: weights,
        modifiedWeights: weights,
        weightsModified: make(map[string]bool),
        allWeightNames: []string{
            "Effort", "HandSwitch", "SameFinger", "SameFingerJump", "Inroll", "Outroll",
            "SHB", "SFB", "HVB", "FVB", "HDB", "FDB", "HFB", "HSB", "FSB", "LSB", "SRB",
            "HDI", "FDI", "D18", "D27", "D36", "D45", "TotalEffortNorm",
            "HSBStrictMode", "FSBStrictMode", "LSBStrictMode",
            "MaxRowEffort1", "MaxRowEffort2", "MaxRowEffort3",
            "RowPenalty1", "RowPenalty2", "RowPenalty3",
        },
        originalBigramIndividualCoeffs: []BigramIndividualCoeff{},
        modifiedBigramIndividualCoeffs: []BigramIndividualCoeff{},
        bigramCoeffsModified: false,
    }

    // Инициализируем все поля как неизмененные
    for _, name := range tracker.allWeightNames {
        tracker.weightsModified[name] = false
    }

    return tracker
}

// MarkWeightModified отмечает, что вес был изменен
func (ct *ConfigChangeTracker) MarkWeightModified(weightName string) {
    ct.weightsModified[weightName] = true
}

// IsWeightModified проверяет, был ли вес изменен
func (ct *ConfigChangeTracker) IsWeightModified(weightName string) bool {
    modified, exists := ct.weightsModified[weightName]
    return exists && modified
}

// GetModifiedWeights возвращает только измененные веса
func (ct *ConfigChangeTracker) GetModifiedWeights() WeightConfig {
    return ct.modifiedWeights
}

// GetAllWeights возвращает все веса (оригинальные или измененные)
func (ct *ConfigChangeTracker) GetAllWeights() WeightConfig {
    return ct.modifiedWeights
}

// SetWeight устанавливает новое значение веса
func (ct *ConfigChangeTracker) SetWeight(weightName string, value float64) {
    switch weightName {
    case "Effort":
        ct.modifiedWeights.Effort = value
    case "HandSwitch":
        ct.modifiedWeights.HandSwitch = value
    case "SameFinger":
        ct.modifiedWeights.SameFinger = value
    case "SameFingerJump":
        ct.modifiedWeights.SameFingerJump = value
    case "Inroll":
        ct.modifiedWeights.Inroll = value
    case "Outroll":
        ct.modifiedWeights.Outroll = value
    case "SHB":
        ct.modifiedWeights.SHB = value
    case "SFB":
        ct.modifiedWeights.SFB = value
    case "HVB":
        ct.modifiedWeights.HVB = value
    case "FVB":
        ct.modifiedWeights.FVB = value
    case "HDB":
        ct.modifiedWeights.HDB = value
    case "FDB":
        ct.modifiedWeights.FDB = value
    case "HFB":
        ct.modifiedWeights.HFB = value
    case "HSB":
        ct.modifiedWeights.HSB = value
    case "FSB":
        ct.modifiedWeights.FSB = value
    case "LSB":
        ct.modifiedWeights.LSB = value
    case "SRB":
        ct.modifiedWeights.SRB = value
    case "HDI":
        ct.modifiedWeights.HDI = value
    case "FDI":
        ct.modifiedWeights.FDI = value
    case "D18":
        ct.modifiedWeights.D18 = value
    case "D27":
        ct.modifiedWeights.D27 = value
    case "D36":
        ct.modifiedWeights.D36 = value
    case "D45":
        ct.modifiedWeights.D45 = value
    case "TotalEffortNorm":
        ct.modifiedWeights.TotalEffortNorm = value
    case "MaxRowEffort1":
        ct.modifiedWeights.MaxRowEffort1 = value
    case "MaxRowEffort2":
        ct.modifiedWeights.MaxRowEffort2 = value
    case "MaxRowEffort3":
        ct.modifiedWeights.MaxRowEffort3 = value
    case "RowPenalty1":
        ct.modifiedWeights.RowPenalty1 = value
    case "RowPenalty2":
        ct.modifiedWeights.RowPenalty2 = value
    case "RowPenalty3":
        ct.modifiedWeights.RowPenalty3 = value
    }
    ct.MarkWeightModified(weightName)
}

// SetIntWeight устанавливает новое целочисленное значение веса
func (ct *ConfigChangeTracker) SetIntWeight(weightName string, value int) {
    switch weightName {
    case "HSBStrictMode":
        ct.modifiedWeights.HSBStrictMode = value
    case "FSBStrictMode":
        ct.modifiedWeights.FSBStrictMode = value
    case "LSBStrictMode":
        ct.modifiedWeights.LSBStrictMode = value
    }
    ct.MarkWeightModified(weightName)
}

// ApplyToConfig применяет измененные веса к конфигурации
func (ct *ConfigChangeTracker) ApplyToConfig(config *KeyboardConfig) {
    // Применяем измененные веса к конфигурации
    if ct.IsWeightModified("Effort") {
        config.Weights.Effort = ct.modifiedWeights.Effort
    }
    if ct.IsWeightModified("HandSwitch") {
        config.Weights.HandSwitch = ct.modifiedWeights.HandSwitch
    }
    if ct.IsWeightModified("SameFinger") {
        config.Weights.SameFinger = ct.modifiedWeights.SameFinger
    }
    if ct.IsWeightModified("SameFingerJump") {
        config.Weights.SameFingerJump = ct.modifiedWeights.SameFingerJump
    }
    if ct.IsWeightModified("Inroll") {
        config.Weights.Inroll = ct.modifiedWeights.Inroll
    }
    if ct.IsWeightModified("Outroll") {
        config.Weights.Outroll = ct.modifiedWeights.Outroll
    }
    if ct.IsWeightModified("SHB") {
        config.Weights.SHB = ct.modifiedWeights.SHB
    }
    if ct.IsWeightModified("SFB") {
        config.Weights.SFB = ct.modifiedWeights.SFB
    }
    if ct.IsWeightModified("HVB") {
        config.Weights.HVB = ct.modifiedWeights.HVB
    }
    if ct.IsWeightModified("FVB") {
        config.Weights.FVB = ct.modifiedWeights.FVB
    }
    if ct.IsWeightModified("HDB") {
        config.Weights.HDB = ct.modifiedWeights.HDB
    }
    if ct.IsWeightModified("FDB") {
        config.Weights.FDB = ct.modifiedWeights.FDB
    }
    if ct.IsWeightModified("HFB") {
        config.Weights.HFB = ct.modifiedWeights.HFB
    }
    if ct.IsWeightModified("HSB") {
        config.Weights.HSB = ct.modifiedWeights.HSB
    }
    if ct.IsWeightModified("FSB") {
        config.Weights.FSB = ct.modifiedWeights.FSB
    }
    if ct.IsWeightModified("LSB") {
        config.Weights.LSB = ct.modifiedWeights.LSB
    }
    if ct.IsWeightModified("SRB") {
        config.Weights.SRB = ct.modifiedWeights.SRB
    }
    if ct.IsWeightModified("HDI") {
        config.Weights.HDI = ct.modifiedWeights.HDI
    }
    if ct.IsWeightModified("FDI") {
        config.Weights.FDI = ct.modifiedWeights.FDI
    }
    if ct.IsWeightModified("D18") {
        config.Weights.D18 = ct.modifiedWeights.D18
    }
    if ct.IsWeightModified("D27") {
        config.Weights.D27 = ct.modifiedWeights.D27
    }
    if ct.IsWeightModified("D36") {
        config.Weights.D36 = ct.modifiedWeights.D36
    }
    if ct.IsWeightModified("D45") {
        config.Weights.D45 = ct.modifiedWeights.D45
    }
    if ct.IsWeightModified("TotalEffortNorm") {
        config.Weights.TotalEffortNorm = ct.modifiedWeights.TotalEffortNorm
    }
    if ct.IsWeightModified("HSBStrictMode") {
        config.Weights.HSBStrictMode = ct.modifiedWeights.HSBStrictMode
    }
    if ct.IsWeightModified("FSBStrictMode") {
        config.Weights.FSBStrictMode = ct.modifiedWeights.FSBStrictMode
    }
    if ct.IsWeightModified("LSBStrictMode") {
        config.Weights.LSBStrictMode = ct.modifiedWeights.LSBStrictMode
    }
    if ct.IsWeightModified("MaxRowEffort1") {
        config.Weights.MaxRowEffort1 = ct.modifiedWeights.MaxRowEffort1
        config.MaxRowEfforts[0] = ct.modifiedWeights.MaxRowEffort1
    }
    if ct.IsWeightModified("MaxRowEffort2") {
        config.Weights.MaxRowEffort2 = ct.modifiedWeights.MaxRowEffort2
        config.MaxRowEfforts[1] = ct.modifiedWeights.MaxRowEffort2
    }
    if ct.IsWeightModified("MaxRowEffort3") {
        config.Weights.MaxRowEffort3 = ct.modifiedWeights.MaxRowEffort3
        config.MaxRowEfforts[2] = ct.modifiedWeights.MaxRowEffort3
    }
    if ct.IsWeightModified("RowPenalty1") {
        config.Weights.RowPenalty1 = ct.modifiedWeights.RowPenalty1
        config.RowEffortPenalties[0] = ct.modifiedWeights.RowPenalty1
    }
    if ct.IsWeightModified("RowPenalty2") {
        config.Weights.RowPenalty2 = ct.modifiedWeights.RowPenalty2
        config.RowEffortPenalties[1] = ct.modifiedWeights.RowPenalty2
    }
    if ct.IsWeightModified("RowPenalty3") {
        config.Weights.RowPenalty3 = ct.modifiedWeights.RowPenalty3
        config.RowEffortPenalties[2] = ct.modifiedWeights.RowPenalty3
    }

    // Применяем измененные индивидуальные коэффициенты биграмм
    if ct.bigramCoeffsModified {
        config.BigramIndividualCoeffs = ct.modifiedBigramIndividualCoeffs
    }
}

// UpdateBaseConfig обновляет базовую конфигурацию, но сохраняет информацию об изменениях
func (ct *ConfigChangeTracker) UpdateBaseConfig(newWeights WeightConfig) {
    // Сохраняем значения измененных параметров
    modifiedValues := make(map[string]interface{})
    for _, name := range ct.allWeightNames {
        if ct.IsWeightModified(name) {
            // Получаем значение измененного параметра
            switch name {
            case "Effort":
                modifiedValues[name] = ct.modifiedWeights.Effort
            case "HandSwitch":
                modifiedValues[name] = ct.modifiedWeights.HandSwitch
            case "SameFinger":
                modifiedValues[name] = ct.modifiedWeights.SameFinger
            case "SameFingerJump":
                modifiedValues[name] = ct.modifiedWeights.SameFingerJump
            case "Inroll":
                modifiedValues[name] = ct.modifiedWeights.Inroll
            case "Outroll":
                modifiedValues[name] = ct.modifiedWeights.Outroll
            case "SHB":
                modifiedValues[name] = ct.modifiedWeights.SHB
            case "SFB":
                modifiedValues[name] = ct.modifiedWeights.SFB
            case "HVB":
                modifiedValues[name] = ct.modifiedWeights.HVB
            case "FVB":
                modifiedValues[name] = ct.modifiedWeights.FVB
            case "HDB":
                modifiedValues[name] = ct.modifiedWeights.HDB
            case "FDB":
                modifiedValues[name] = ct.modifiedWeights.FDB
            case "HFB":
                modifiedValues[name] = ct.modifiedWeights.HFB
            case "HSB":
                modifiedValues[name] = ct.modifiedWeights.HSB
            case "FSB":
                modifiedValues[name] = ct.modifiedWeights.FSB
            case "LSB":
                modifiedValues[name] = ct.modifiedWeights.LSB
            case "SRB":
                modifiedValues[name] = ct.modifiedWeights.SRB
            case "HDI":
                modifiedValues[name] = ct.modifiedWeights.HDI
            case "FDI":
                modifiedValues[name] = ct.modifiedWeights.FDI
            case "D18":
                modifiedValues[name] = ct.modifiedWeights.D18
            case "D27":
                modifiedValues[name] = ct.modifiedWeights.D27
            case "D36":
                modifiedValues[name] = ct.modifiedWeights.D36
            case "D45":
                modifiedValues[name] = ct.modifiedWeights.D45
            case "TotalEffortNorm":
                modifiedValues[name] = ct.modifiedWeights.TotalEffortNorm
            case "HSBStrictMode":
                modifiedValues[name] = ct.modifiedWeights.HSBStrictMode
            case "FSBStrictMode":
                modifiedValues[name] = ct.modifiedWeights.FSBStrictMode
            case "LSBStrictMode":
                modifiedValues[name] = ct.modifiedWeights.LSBStrictMode
            case "MaxRowEffort1":
                modifiedValues[name] = ct.modifiedWeights.MaxRowEffort1
            case "MaxRowEffort2":
                modifiedValues[name] = ct.modifiedWeights.MaxRowEffort2
            case "MaxRowEffort3":
                modifiedValues[name] = ct.modifiedWeights.MaxRowEffort3
            case "RowPenalty1":
                modifiedValues[name] = ct.modifiedWeights.RowPenalty1
            case "RowPenalty2":
                modifiedValues[name] = ct.modifiedWeights.RowPenalty2
            case "RowPenalty3":
                modifiedValues[name] = ct.modifiedWeights.RowPenalty3
            }
        }
    }

    // Сохраняем индивидуальные коэффициенты биграмм, если они были изменены
    var modifiedBigramCoeffs []BigramIndividualCoeff
    if ct.bigramCoeffsModified {
        modifiedBigramCoeffs = ct.modifiedBigramIndividualCoeffs
    }

    // Обновляем оригинальные веса
    ct.originalWeights = newWeights

    // Восстанавливаем измененные значения в новой конфигурации
    for name, value := range modifiedValues {
        switch name {
        case "Effort":
            ct.modifiedWeights.Effort = value.(float64)
        case "HandSwitch":
            ct.modifiedWeights.HandSwitch = value.(float64)
        case "SameFinger":
            ct.modifiedWeights.SameFinger = value.(float64)
        case "SameFingerJump":
            ct.modifiedWeights.SameFingerJump = value.(float64)
        case "Inroll":
            ct.modifiedWeights.Inroll = value.(float64)
        case "Outroll":
            ct.modifiedWeights.Outroll = value.(float64)
        case "SHB":
            ct.modifiedWeights.SHB = value.(float64)
        case "SFB":
            ct.modifiedWeights.SFB = value.(float64)
        case "HVB":
            ct.modifiedWeights.HVB = value.(float64)
        case "FVB":
            ct.modifiedWeights.FVB = value.(float64)
        case "HDB":
            ct.modifiedWeights.HDB = value.(float64)
        case "FDB":
            ct.modifiedWeights.FDB = value.(float64)
        case "HFB":
            ct.modifiedWeights.HFB = value.(float64)
        case "HSB":
            ct.modifiedWeights.HSB = value.(float64)
        case "FSB":
            ct.modifiedWeights.FSB = value.(float64)
        case "LSB":
            ct.modifiedWeights.LSB = value.(float64)
        case "SRB":
            ct.modifiedWeights.SRB = value.(float64)
        case "HDI":
            ct.modifiedWeights.HDI = value.(float64)
        case "FDI":
            ct.modifiedWeights.FDI = value.(float64)
        case "D18":
            ct.modifiedWeights.D18 = value.(float64)
        case "D27":
            ct.modifiedWeights.D27 = value.(float64)
        case "D36":
            ct.modifiedWeights.D36 = value.(float64)
        case "D45":
            ct.modifiedWeights.D45 = value.(float64)
        case "TotalEffortNorm":
            ct.modifiedWeights.TotalEffortNorm = value.(float64)
        case "HSBStrictMode":
            ct.modifiedWeights.HSBStrictMode = value.(int)
        case "FSBStrictMode":
            ct.modifiedWeights.FSBStrictMode = value.(int)
        case "LSBStrictMode":
            ct.modifiedWeights.LSBStrictMode = value.(int)
        case "MaxRowEffort1":
            ct.modifiedWeights.MaxRowEffort1 = value.(float64)
        case "MaxRowEffort2":
            ct.modifiedWeights.MaxRowEffort2 = value.(float64)
        case "MaxRowEffort3":
            ct.modifiedWeights.MaxRowEffort3 = value.(float64)
        case "RowPenalty1":
            ct.modifiedWeights.RowPenalty1 = value.(float64)
        case "RowPenalty2":
            ct.modifiedWeights.RowPenalty2 = value.(float64)
        case "RowPenalty3":
            ct.modifiedWeights.RowPenalty3 = value.(float64)
        }
    }

    // Восстанавливаем измененные индивидуальные коэффициенты биграмм
    if len(modifiedBigramCoeffs) > 0 {
        ct.modifiedBigramIndividualCoeffs = modifiedBigramCoeffs
        ct.bigramCoeffsModified = true
    } else {
        ct.bigramCoeffsModified = false
        ct.originalBigramIndividualCoeffs = []BigramIndividualCoeff{}
        ct.modifiedBigramIndividualCoeffs = []BigramIndividualCoeff{}
    }
}

// SetBigramIndividualCoeffs устанавливает новые индивидуальные коэффициенты биграмм
func (ct *ConfigChangeTracker) SetBigramIndividualCoeffs(coeffs []BigramIndividualCoeff) {
    ct.modifiedBigramIndividualCoeffs = coeffs
    ct.bigramCoeffsModified = true
}

// GetAllModifiedParams возвращает все измененные параметры
func (ct *ConfigChangeTracker) GetAllModifiedParams() map[string]interface{} {
    modifiedParams := make(map[string]interface{})
    for _, name := range ct.allWeightNames {
        if ct.IsWeightModified(name) {
            switch name {
            case "Effort":
                modifiedParams[name] = ct.modifiedWeights.Effort
            case "HandSwitch":
                modifiedParams[name] = ct.modifiedWeights.HandSwitch
            case "SameFinger":
                modifiedParams[name] = ct.modifiedWeights.SameFinger
            case "SameFingerJump":
                modifiedParams[name] = ct.modifiedWeights.SameFingerJump
            case "Inroll":
                modifiedParams[name] = ct.modifiedWeights.Inroll
            case "Outroll":
                modifiedParams[name] = ct.modifiedWeights.Outroll
            case "SHB":
                modifiedParams[name] = ct.modifiedWeights.SHB
            case "SFB":
                modifiedParams[name] = ct.modifiedWeights.SFB
            case "HVB":
                modifiedParams[name] = ct.modifiedWeights.HVB
            case "FVB":
                modifiedParams[name] = ct.modifiedWeights.FVB
            case "HDB":
                modifiedParams[name] = ct.modifiedWeights.HDB
            case "FDB":
                modifiedParams[name] = ct.modifiedWeights.FDB
            case "HFB":
                modifiedParams[name] = ct.modifiedWeights.HFB
            case "HSB":
                modifiedParams[name] = ct.modifiedWeights.HSB
            case "FSB":
                modifiedParams[name] = ct.modifiedWeights.FSB
            case "LSB":
                modifiedParams[name] = ct.modifiedWeights.LSB
            case "SRB":
                modifiedParams[name] = ct.modifiedWeights.SRB
            case "HDI":
                modifiedParams[name] = ct.modifiedWeights.HDI
            case "FDI":
                modifiedParams[name] = ct.modifiedWeights.FDI
            case "D18":
                modifiedParams[name] = ct.modifiedWeights.D18
            case "D27":
                modifiedParams[name] = ct.modifiedWeights.D27
            case "D36":
                modifiedParams[name] = ct.modifiedWeights.D36
            case "D45":
                modifiedParams[name] = ct.modifiedWeights.D45
            case "TotalEffortNorm":
                modifiedParams[name] = ct.modifiedWeights.TotalEffortNorm
            case "HSBStrictMode":
                modifiedParams[name] = ct.modifiedWeights.HSBStrictMode
            case "FSBStrictMode":
                modifiedParams[name] = ct.modifiedWeights.FSBStrictMode
            case "LSBStrictMode":
                modifiedParams[name] = ct.modifiedWeights.LSBStrictMode
            case "MaxRowEffort1":
                modifiedParams[name] = ct.modifiedWeights.MaxRowEffort1
            case "MaxRowEffort2":
                modifiedParams[name] = ct.modifiedWeights.MaxRowEffort2
            case "MaxRowEffort3":
                modifiedParams[name] = ct.modifiedWeights.MaxRowEffort3
            case "RowPenalty1":
                modifiedParams[name] = ct.modifiedWeights.RowPenalty1
            case "RowPenalty2":
                modifiedParams[name] = ct.modifiedWeights.RowPenalty2
            case "RowPenalty3":
                modifiedParams[name] = ct.modifiedWeights.RowPenalty3
            }
        }
    }

    // Добавляем информацию об измененных индивидуальных коэффициентах биграмм
    if ct.bigramCoeffsModified {
        modifiedParams["BigramIndividualCoeffs"] = ct.modifiedBigramIndividualCoeffs
    }

    return modifiedParams
}

// ResetModifiedParams сбрасывает все изменения параметров
func (ct *ConfigChangeTracker) ResetModifiedParams() {
    // Сбрасываем все флаги изменений
    for _, name := range ct.allWeightNames {
        ct.weightsModified[name] = false
    }

    // Сбрасываем флаг изменения индивидуальных коэффициентов биграмм
    ct.bigramCoeffsModified = false

    // Восстанавливаем оригинальные значения
    ct.modifiedWeights = ct.originalWeights
    ct.modifiedBigramIndividualCoeffs = ct.originalBigramIndividualCoeffs
}