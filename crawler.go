package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
)


type crawler struct {
	downloadTimeout   time.Duration
	collectionTimeout time.Duration
	timeBetweenSteps  time.Duration
	year              string
	month             string
	output            string
}

func (c crawler) crawl() ([]string, error) {
	// Pegar variáveis de ambiente

	// Chromedp setup.
	log.SetOutput(os.Stderr) // Enviando logs para o stderr para não afetar a execução do coletor.

	alloc, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true), // mude para false para executar com navegador visível.
			chromedp.NoSandbox,
			chromedp.DisableGPU,
		)...,
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(
		alloc,
		chromedp.WithLogf(log.Printf), // remover comentário para depurar
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, c.collectionTimeout)
	defer cancel()

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser MPRO.

	// Contracheque
	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.abreCaixaDialogo(ctx, "contra"); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)
	if err := c.exportaPlanilha(ctx, cqFname); err != nil {
		log.Fatalf("Erro fazendo download do contracheque: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.abreCaixaDialogo(ctx, "inde"); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")
	iFname := c.downloadFilePath("indenizatorias")
	log.Printf("Fazendo download das indenizações (%s)...", iFname)
	if err := c.exportaPlanilha(ctx, iFname); err != nil {
		log.Fatalf("Erro fazendo download dos indenizações: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")


	// Retorna caminhos completos dos arquivos baixados.
	return []string{cqFname, iFname}, nil
}

func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("membros-ativos-%s-%s-%s.csv", prefix, c.month, c.year))
}

func (c crawler) abreCaixaDialogo(ctx context.Context, tipo string) error {
	var concatenated string
	if tipo == "contra"{
		const (
			baseURL  = "https://servicos-portal.mpro.mp.br/plcVis/frameset?__report=..%2FROOT%2Frel%2Fcontracheque%2Fmembros%2FremuneracaoMembrosAtivos.rptdesign&anomes="
			finalURL = "&nome=&cargo=&lotacao="
		)
		concatenated = fmt.Sprintf("%s%s%s%s", baseURL, c.year, c.month, finalURL)
	} else {
		const (
			baseURL  = "https://servicos-portal.mpro.mp.br/plcVis/frameset?__report=..%2FROOT%2Frel%2Fcontracheque%2Fmembros%2FverbasIndenizatoriasMembrosAtivos.rptdesign&anomes="
		)
		concatenated = fmt.Sprintf("%s%s%s", baseURL, c.year, c.month)
	}

	return chromedp.Run(ctx,
		chromedp.Navigate(concatenated),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona caixa de dialogo
		chromedp.Click(`//*[@title='Exportar dados']`, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona as colunas
		chromedp.Click(`/html/body/table/tbody/tr[4]/td[1]/div[1]/div[2]/div/div[1]/table/tbody/tr[5]/td[2]/table/tbody/tr/td/table/tbody/tr[1]/td/input`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
		
		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),
	)
}

// exportaPlanilha clica no botão correto para exportar para excel, espera um tempo para download renomeia o arquivo.
func (c crawler) exportaPlanilha(ctx context.Context, fName string) error {
	chromedp.Run(ctx,
		// Clica no botão de download 
		chromedp.Click(`/html/body/table/tbody/tr[4]/td[1]/div[1]/div[2]/div/div[2]/div[2]/div/div[1]/input`, chromedp.BySearch, chromedp.NodeVisible),
	)
	time.Sleep(c.downloadTimeout)

	if err := nomeiaDownload(c.output, fName); err != nil {
		return fmt.Errorf("erro renomeando arquivo (%s): %v", fName, err)
	}
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return fmt.Errorf("download do arquivo de %s não realizado", fName)
	}
	return nil
}

// nomeiaDownload dá um nome ao último arquivo modificado dentro do diretório
// passado como parâmetro nomeiaDownload dá pega um arquivo
func nomeiaDownload(output, fName string) error {
	// Identifica qual foi o ultimo arquivo
	files, err := os.ReadDir(output)
	if err != nil {
		return fmt.Errorf("erro lendo diretório %s: %v", output, err)
	}
	var newestFPath string
	var newestTime int64 = 0
	for _, f := range files {
		fPath := filepath.Join(output, f.Name())
		fi, err := os.Stat(fPath)
		if err != nil {
			return fmt.Errorf("erro obtendo informações sobre arquivo %s: %v", fPath, err)
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFPath = fPath
		}
	}
	// Renomeia o ultimo arquivo modificado.
	if err := os.Rename(newestFPath, fName); err != nil {
		return fmt.Errorf("erro renomeando último arquivo modificado (%s)->(%s): %v", newestFPath, fName, err)
	}
	return nil
}