package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type confSpec struct {
	Month string
}

const (
	defaultFileDownloadTimeout = 20 * time.Second // Duração que o coletor deve esperar até que o download de cada um dos arquivos seja concluído
	defaultGeneralTimeout      = 6 * time.Minute  // Duração máxima total da coleta de todos os arquivos. Valor padrão calculado a partir de uma média de execuções ~4.5min
	defaulTimeBetweenSteps     = 7 * time.Second //Tempo de espera entre passos do coletor."
)

func main() {
	if _, err := strconv.Atoi(os.Getenv("MONTH")); err != nil {
		log.Fatalf("Invalid month (\"%s\"): %q", os.Getenv("MONTH"), err)
	}
	month := os.Getenv("MONTH")

	if _, err := strconv.Atoi(os.Getenv("YEAR")); err != nil {
		log.Fatalf("Invalid year (\"%s\"): %q", os.Getenv("YEAR"), err)
	}
	year := os.Getenv("YEAR")

	outputFolder := os.Getenv("OUTPUT_FOLDER")
	if outputFolder == "" {
		outputFolder = "./output"
	}

	if err := os.Mkdir(outputFolder, os.ModePerm); err != nil && !os.IsExist(err) {
		log.Fatalf("Error creating output folder(%s): %q", outputFolder, err)
	}

	downloadTimeout := defaultFileDownloadTimeout
	if os.Getenv("DOWNLOAD_TIMEOUT") != "" {
		var err error
		downloadTimeout, err = time.ParseDuration(os.Getenv("DOWNLOAD_TIMEOUT"))
		if err != nil {
			log.Fatalf("Invalid DOWNLOAD_TIMEOUT (\"%s\"): %q", os.Getenv("DOWNLOAD_TIMEOUT"), err)
		}
	}

	generalTimeout := defaultGeneralTimeout
	if os.Getenv("GENERAL_TIMEOUT") != "" {
		var err error
		generalTimeout, err = time.ParseDuration(os.Getenv("GENERAL_TIMEOUT"))
		if err != nil {
			log.Fatalf("Invalid GENERAL_TIMEOUT (\"%s\"): %q", os.Getenv("GENERAL_TIMEOUT"), err)
		}
	}

	timeBetweenSteps := defaulTimeBetweenSteps
	if os.Getenv("TIME_BETWEEN_STEPS") != "" {
		var err error
		timeBetweenSteps, err = time.ParseDuration(os.Getenv("TIME_BETWEEN_STEPS"))
		if err != nil {
			log.Fatalf("Invalid TIME_BETWEEN_STEPS (\"%s\"): %q", os.Getenv("TIME_BETWEEN_STEPS"), err)
		}
	}
	c := crawler{
		downloadTimeout:   downloadTimeout,
		collectionTimeout: generalTimeout,
		timeBetweenSteps:  timeBetweenSteps,
		year:              year,
		month:             month,
		output:            outputFolder,
	}
	downloads, err := c.crawl()
	if err != nil {
		log.Fatalf("Error crawling (%s, %s, %s): %v", year, month, outputFolder, err)
	}

	// O parser do CNJ espera os arquivos separados por \n. Mudanças aqui tem
	// refletir as expectativas lá.
	fmt.Println(strings.Join(downloads, "\n"))
}