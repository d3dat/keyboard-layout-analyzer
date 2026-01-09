package main

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// ProcessTextFile processes a text file to generate language statistics
func ProcessTextFile(textFile, alphabetString, outputFile string) error {
	// Parse the alphabet string to handle special cases
	alphabet, charGroups := parseAlphabet(alphabetString)

	// Debug: Print the parsed alphabet
	// fmt.Printf("Parsed alphabet: %+v\n", alphabet)
	// fmt.Printf("Parsed charGroups: %+v\n", charGroups)

	// Read the text file
	content, err := os.ReadFile(textFile)
	if err != nil {
		return fmt.Errorf("error reading text file: %v", err)
	}

	// Convert to lowercase for processing
	text := strings.ToLower(string(content))

	// Get all unique characters that will be in the final output (after applying character groups)
	uniqueChars := make(map[string]bool)

	// Add all characters from the original alphabet to uniqueChars, but map them through character groups
	for r := range alphabet {
		charStr := string(r)
		mappedChar := mapToCharacterGroup(charStr, charGroups)
		uniqueChars[mappedChar] = true
	}

	// Initialize counts
	bigramCounts := make(map[string]int)
	unigramCounts := make(map[string]int)

	// Initialize unigram counts with 0 for all characters in alphabet
	for char := range uniqueChars {
		unigramCounts[char] = 0
	}

	// Split text into words based on non-alphabet characters
	// We need to preserve spaces between words while removing other punctuation
	// Using a more comprehensive regex to handle Cyrillic and other Unicode characters
	re := regexp.MustCompile(`[^\p{L}\p{N}_ ]+`) // \p{L} matches any Unicode letter, \p{N} matches any Unicode number
	textWithoutPunct := re.ReplaceAllString(text, " ")

	// Split text into words based on spaces
	words := strings.Fields(textWithoutPunct)

	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		// Process the word character by character
		cleanWord := ""
		for _, char := range word {
			// Check if character is in our alphabet or is a space
			if isInAlphabet(char, alphabet, charGroups) {
				cleanWord += string(char)
			}
		}

		// Process the cleaned word
		if len(cleanWord) > 0 {
			// Convert to runes to properly handle Unicode characters
			runes := []rune(cleanWord)

			// Count unigrams
			for _, char := range runes {
				charStr := string(char)
				// Handle character groups - map to the first character in the group
				mappedChar := mapToCharacterGroup(charStr, charGroups)
				unigramCounts[mappedChar]++
			}

			// Count bigrams
			for i := 0; i < len(runes)-1; i++ {
				char1 := string(runes[i])
				char2 := string(runes[i+1])

				// Map to character groups if needed
				char1 = mapToCharacterGroup(char1, charGroups)
				char2 = mapToCharacterGroup(char2, charGroups)

				// Skip if either character is not in alphabet
				// Convert string back to rune to check in alphabet
				rune1 := []rune(char1)[0]
				rune2 := []rune(char2)[0]
				inAlphabet1 := isInAlphabet(rune1, alphabet, charGroups)
				inAlphabet2 := isInAlphabet(rune2, alphabet, charGroups)
				if inAlphabet1 && inAlphabet2 {
					bigram := char1 + char2
					bigramCounts[bigram]++
				}
			}
		}
	}

	// Calculate frequencies
	totalBigrams := 0
	for _, count := range bigramCounts {
		totalBigrams += count
	}

	totalUnigrams := 0
	for _, count := range unigramCounts {
		totalUnigrams += count
	}

	// Initialize all possible bigrams and unigrams with 0 frequency
	bigramFreqs := make(map[string]float64)
	unigramFreqs := make(map[string]float64)

	// Initialize all possible bigrams with 0
	for char1 := range uniqueChars {
		for char2 := range uniqueChars {
			bigram := char1 + char2
			bigramFreqs[bigram] = 0.0
		}
	}

	// Initialize all possible unigrams with 0
	for char := range uniqueChars {
		unigramFreqs[char] = 0.0
	}

	// Convert counts to frequencies for found bigrams and unigrams
	if totalBigrams > 0 {
		for bigram, count := range bigramCounts {
			// Only update if the bigram was found in text
			bigramFreqs[bigram] = float64(count) / float64(totalBigrams)
			// Debug: fmt.Printf("Setting frequency for bigram %s: %f\n", bigram, bigramFreqs[bigram])
		}
	}

	if totalUnigrams > 0 {
		for char, count := range unigramCounts {
			// Only update if the unigram was found in text
			unigramFreqs[char] = float64(count) / float64(totalUnigrams)
			// Debug: fmt.Printf("Setting frequency for unigram %s: %f\n", char, unigramFreqs[char])
		}
	}

	// Write to output file manually to ensure proper ordering
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer file.Close()

	// Write the JSON structure manually with proper formatting
	fmt.Fprintf(file, "{\n")
	fmt.Fprintf(file, "  \"language\": \"Generated from text file\",\n")

	// Write characters in sorted order
	fmt.Fprintf(file, "  \"characters\": {\n")
	charPairs := getSortedPairs(unigramFreqs)
	for i, pair := range charPairs {
		if i == len(charPairs)-1 {
			fmt.Fprintf(file, "    \"%s\": %g\n", pair.Key, pair.Value)
		} else {
			fmt.Fprintf(file, "    \"%s\": %g,\n", pair.Key, pair.Value)
		}
	}
	fmt.Fprintf(file, "  },\n")

	// Write bigrams in sorted order
	fmt.Fprintf(file, "  \"bigrams\": {\n")
	bigramPairs := getSortedPairs(bigramFreqs)
	for i, pair := range bigramPairs {
		if i == len(bigramPairs)-1 {
			fmt.Fprintf(file, "    \"%s\": %g\n", pair.Key, pair.Value)
		} else {
			fmt.Fprintf(file, "    \"%s\": %g,\n", pair.Key, pair.Value)
		}
	}
	fmt.Fprintf(file, "  }\n")
	fmt.Fprintf(file, "}\n")

	fmt.Printf("Обработка файла %s завершена, результаты записаны в файл %s\n", textFile, outputFile)

	return nil
}

// KeyValue represents a key-value pair for sorting
type KeyValue struct {
	Key   string
	Value float64
}

// getSortedPairs returns a slice of key-value pairs sorted by frequency (descending) and alphabetically for equal frequencies
func getSortedPairs(input map[string]float64) []KeyValue {
	// Create a slice of key-value pairs
	pairs := make([]KeyValue, 0, len(input))
	for k, v := range input {
		pairs = append(pairs, KeyValue{Key: k, Value: v})
	}

	// Sort pairs by frequency (descending) and alphabetically for equal frequencies
	sort.Slice(pairs, func(i, j int) bool {
		freqI := pairs[i].Value
		freqJ := pairs[j].Value
		if freqI == freqJ {
			return pairs[i].Key < pairs[j].Key // Alphabetical order for equal frequencies
		}
		return freqI > freqJ // Descending order by frequency
	})

	return pairs
}

// parseAlphabet parses the alphabet string and handles special cases
func parseAlphabet(alphabetString string) (map[rune]bool, map[string]string) {
	alphabet := make(map[rune]bool)
	charGroups := make(map[string]string) // Maps equivalent chars to the first char in the group

	// Convert string to runes to properly handle Unicode characters
	runes := []rune(alphabetString)
	i := 0
	for i < len(runes) {
		char := runes[i]

		// Handle underscore as space
		if char == '_' {
			alphabet[' '] = true
			i++
			continue
		}

		// Handle square bracket groups
		if char == '[' {
			// Find the closing bracket
			endBracket := -1
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == ']' {
					endBracket = j
					break
				}
			}

			if endBracket != -1 {
				// Extract characters inside brackets
				// Need to get the runes between brackets
				groupRunes := runes[i+1 : endBracket]
				if len(groupRunes) > 0 {
					firstChar := string(groupRunes[0])
					for _, groupChar := range groupRunes {
						charGroups[string(groupChar)] = firstChar
						// Add the first character to the alphabet
						alphabet[groupRunes[0]] = true
					}
				}
				i = endBracket + 1
				continue
			}
		}

		// Handle escape sequences
		if char == '\\' && i+1 < len(runes) {
			nextChar := runes[i+1]
			// Handle escaped characters: \[, \], \\
			if nextChar == '[' || nextChar == ']' || nextChar == '\\' {
				alphabet[nextChar] = true
				i += 2 // Skip both the backslash and the next character
				continue
			} else {
				// If it's not a recognized escape sequence, treat backslash as a regular character
				alphabet[char] = true
				i++
				continue
			}
		}

		// Regular character (but skip spaces since they shouldn't be part of the alphabet)
		if char != ' ' {
			alphabet[char] = true
		}
		i++
	}

	return alphabet, charGroups
}

// isInAlphabet checks if a character is in the alphabet
func isInAlphabet(char rune, alphabet map[rune]bool, charGroups map[string]string) bool {
	// Check if the character is directly in the alphabet
	if alphabet[char] {
		return true
	}

	// Check if it's part of a character group
	charStr := string(char)
	if mappedChar, exists := charGroups[charStr]; exists {
		mappedRune := rune(mappedChar[0])
		result := alphabet[mappedRune]
		return result
	}

	return false
}

// mapToCharacterGroup maps a character to its group representative
func mapToCharacterGroup(char string, charGroups map[string]string) string {
	if mappedChar, exists := charGroups[char]; exists {
		return mappedChar
	}
	return char
}

// normalizeSpaces replaces multiple consecutive spaces with a single space
func normalizeSpaces(text string) string {
	// Replace multiple spaces with a single space
	for {
		newText := strings.Replace(text, "  ", " ", -1)
		if newText == text {
			break
		}
		text = newText
	}
	return text
}
