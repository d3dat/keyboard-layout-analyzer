package main

import (
	"flag"
	"fmt"
	"github.com/peterh/liner"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultLangFile   = "language.json"
	defaultConfigFile = "config.txt"
	defaultLayoutFile = "layout.txt"
)

func main() {
	// Определяем новые флаги командной строки
	helpFlag := flag.Bool("h", false, "Показать справку")
	helpLongFlag := flag.Bool("help", false, "Показать справку")
	configFileFlag := flag.String("config", defaultConfigFile, "Имя файла с конфигурацией")
	layoutFileFlag := flag.String("layout", defaultLayoutFile, "Имя файла с раскладками")
	outputFileFlag := flag.String("output", "", "Имя файла для сохранения новых раскладок (если не указано, используется файл из --layout)")
	langFileFlag := flag.String("lang", defaultLangFile, "Имя файла со статистикой букв в языке")
	effortFileFlag := flag.String("effort", "", "Имя файла с матрицей усилий по пальцам (если не указано, используется из конфигурационного файла)")
	textFileFlag := flag.String("text", "", "Имя файла с текстом для генерации языковой статистики")
	alphabetFlag := flag.String("alphabet", "", "Алфавит для генерации языкового файла (все символы из строки рассматриваются как часть алфавита)")

	// Parse флаги
	flag.Parse()

	// Проверяем флаг справки (как короткий, так и длинный)
	if *helpFlag || *helpLongFlag {
		printMainHelp()
		os.Exit(0)
	}

	// Check if we're in text processing mode
	if *textFileFlag != "" {
		// Validate required arguments for text mode
		if *outputFileFlag == "" {
			fmt.Fprintf(os.Stderr, "Error: --output is required when using --text\n")
			printShortHelp()
			os.Exit(1)
		}

		if *alphabetFlag == "" {
			fmt.Fprintf(os.Stderr, "Error: --alphabet is required when using --text\n")
			printShortHelp()
			os.Exit(1)
		}

		// Validate that no other conflicting arguments are present
		if flag.NFlag() > 3 { // text, output, alphabet = 3 flags
			fmt.Fprintf(os.Stderr, "Error: In --text mode, only --text, --output, and --alphabet flags are allowed\n")
			printShortHelp()
			os.Exit(1)
		}

		// Process the text file
		err := ProcessTextFile(*textFileFlag, *alphabetFlag, *outputFileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing text file: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Если программа запускается без аргументов, проверяем наличие файлов по умолчанию
	if flag.NFlag() == 0 && !*helpFlag && !*helpLongFlag {
		// Проверяем наличие файлов по умолчанию
		defaultFiles := []string{defaultConfigFile, defaultLayoutFile, defaultLangFile}
		missingFiles := []string{}

		for _, fileName := range defaultFiles {
			if _, err := os.Stat(fileName); os.IsNotExist(err) {
				missingFiles = append(missingFiles, fileName)
			}
		}

		// Если какие-то файлы отсутствуют, выводим краткую справку и завершаем программу
		if len(missingFiles) > 0 {
			fmt.Fprintf(os.Stderr, "Отсутствуют необходимые файлы: %s\n", strings.Join(missingFiles, ", "))
			printShortHelp()
			os.Exit(1)
		}
	}

	// Получаем рабочую директорию для поиска файлов
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}

	langFile := filepath.Join(workDir, *langFileFlag)
	configFile := filepath.Join(workDir, *configFileFlag)
	layoutFile := filepath.Join(workDir, *layoutFileFlag)

	// Если указан выходной файл, используем его, иначе используем тот же файл что и для загрузки
	var outputFile string
	if *outputFileFlag != "" {
		outputFile = filepath.Join(workDir, *outputFileFlag)
	} else {
		outputFile = layoutFile
	}

	// Загружаем данные
	langData, config, layouts, err := LoadAllData(langFile, configFile, layoutFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки данных: %v\n", err)
		os.Exit(1)
	}

	// Если указан отдельный файл с усилиями, загружаем матрицу усилий из него
	if *effortFileFlag != "" {
		effortMatrix, err := LoadEffortMatrix(*effortFileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка загрузки матрицы усилий: %v\n", err)
			os.Exit(1)
		}

		// Обновляем матрицу усилий в конфигурации
		config.EffortMatrix = effortMatrix
	}

	// Создаём обработчик команд
	handler := NewCommandHandler(langData, config, layouts, langFile, configFile, layoutFile, outputFile, *effortFileFlag)

	// Командный режим (REPL)
	interactiveMode(handler, langFile, configFile, layoutFile)
}

// interactiveMode запускает интерактивный режим с историей команд
func interactiveMode(handler *CommandHandler, langFile, configFile, layoutFile string) {
	line := liner.NewLiner()
	defer line.Close()

	// Включаем историю команд
	line.SetCtrlCAborts(true)

	fmt.Println("Анализатор раскладок сплит-клавиатуры")
	fmt.Println("Введите 'help' для справки по командам")
	fmt.Println()

	for {
		cmd, err := line.Prompt("> ")
		if err != nil {
			// EOF или Ctrl+C
			if err == liner.ErrPromptAborted {
				fmt.Println()
				fmt.Println("До свидания!")
			} else {
				fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
			}
			break
		}

		cmd = strings.TrimSpace(cmd)

		// Пропускаем пустые строки
		if cmd == "" {
			continue
		}

		// Добавляем команду в историю, если она не пустая
		line.AppendHistory(cmd)

		// Обрабатываем специальные команды
		if cmd == "exit" || cmd == "quit" || cmd == "q" {
			fmt.Println("До свидания!")
			break
		}

		// Выполняем команду
		if err := handler.ParseCommand(cmd); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		}

		fmt.Println()
	}
}

// printMainHelp выводит справку по использованию программы
func printMainHelp() {
	helpText := `Анализатор раскладок клавиатуры

Использование:
  kbda [OPTIONS]

Параметры командной строки:
  -h, --help        - Показать справку по использованию программы
  --config FILE     - Указать имя файла с конфигурацией (по умолчанию config.txt)
  --layout FILE     - Указать имя файла с раскладками (по умолчанию layout.txt)
  --lang FILE       - Указать имя файла со статистикой букв в языке (по умолчанию language.json)
  --effort FILE     - Указать имя файла с матрицей усилий по пальцам (если не указано, используется из конфигурационного файла)
  --output FILE     - Указать имя файла для сохранения новых раскладок (если не указано, используется файл из --layout)
  --text FILE       - Указать имя текстового файла для генерации языковой статистики
  --alphabet STRING - Указать алфавит для генерации языковой статистики (все символы из строки рассматриваются как часть алфавита)

Режим генерации языковой статистики:
  kbda --text file.txt --alphabet string --output lang.json
  В этом режиме обязательно должны быть указаны опции --output и --alphabet.
  Формат строки для задания алфавитаж описан в файле README.md.

Примеры:
  kbda                           # Запуск в интерактивном режиме с файлами по умолчанию
  kbda --config myconfig.txt     # Запуск с нестандартным файлом конфигурации
  kbda --lang ru.json --layout my_layout.txt  # Запуск с нестандартными файлами языка и раскладки
  kbda --layout my_layout.txt --output new_layouts.txt  # Запуск с файлом для сохранения новых раскладок
  kbda --text file.txt --alphabet абвг_д --output lang.json  # Генерация языковой статистики из текста

Без аргументов программа переходит в интерактивный режим (REPL) с использованием имен файлов по умолчанию:
  config.txt
  layout.txt
  language.json

Доступные команды в интерактивном режиме:
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

Файлы конфигурации:
  language.json  - Частоты букв и биграмм языка
  config.txt     - Конфигурация клавиатуры (усилия, позиции, веса)
  layout.txt     - Раскладки для анализа
`
	fmt.Print(helpText)
}

// printShortHelp выводит краткую справку по использованию программы
func printShortHelp() {
	shortHelpText := `
Анализатор раскладок клавиатуры

Использование:
  kbda [OPTIONS]

Опции командной строки:
  -h            - Показать полную справку
  --config FILE - Указать имя файла с конфигурацией (по умолчанию config.txt)
  --layout FILE - Указать имя файла с раскладками (по умолчанию layout.txt)
  --lang FILE   - Указать имя файла со статистикой букв в языке (по умолчанию language.json)
  --effort FILE - Указать имя файла с матрицей усилий по пальцам
  --output FILE - Указать имя файла для сохранения новых раскладок
  --text FILE   - Указать имя текстового файла для генерации языковой статистики
  --alphabet STRING - Указать алфавит для генерации языкового файла

Режим генерации языковой статистики:
  kbda --text file.txt --alphabet string --output lang.json
  В этом режиме обязательно должны быть указаны опции --output и --alphabet.
  Наличие в этом режиме других аргументов приведет к ошибке.

Примеры:
  kbda                           # Запуск в интерактивном режиме с файлами по умолчанию
  kbda --config myconfig.txt     # Запуск с нестандартным файлом конфигурации
  kbda --text file.txt --alphabet абвг_д --output lang.json  # Генерация языковой статистики из текста

Для запуска без аргументов необходимы файлы:
  config.txt
  layout.txt
  language.json
`
	fmt.Print(shortHelpText)
}
