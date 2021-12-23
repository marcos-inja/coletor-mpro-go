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

const (
	direitosPessoaisXPATH = "/html/body/div[5]/div/div[31]/div[2]/table/tbody/tr/td"
	indenizacoesXPATH     = "/html/body/div[5]/div/div[25]/div[2]/table/tbody/tr/td"
	verbasXPATH           = "/html/body/div[5]/div/div[28]/div[2]/table/tbody/tr/td"
	controleXPATH         = "/html/body/div[5]/div/div[52]/div[2]/table/tbody/tr/td"
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
			chromedp.Flag("headless", false), // mude para false para executar com navegador visível.
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

	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)

	if err := c.selectionaOrgaoMesAno(ctx); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser CNJ.

	// O contra cheque é a aba padrão, por isso não precisa haver clique.
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)
	if err := c.exportaExcel(ctx, cqFname); err != nil {
		log.Fatalf("Erro fazendo download do contracheque: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	iFname := c.downloadFilePath("indenizacoes")
	log.Printf("Fazendo download das indenizações (%s)...", iFname)
	if err := c.clicaAba(ctx, indenizacoesXPATH); err != nil {
		log.Fatalf("Erro clicando na aba de indenizações: %v", err)
	}
	if err := c.exportaExcel(ctx, iFname); err != nil {
		log.Fatalf("Erro fazendo download dos indenizações: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	

	// Retorna caminhos completos dos arquivos baixados.
	return []string{cqFname, iFname}, nil
}

func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("%s-%s-%s.csv", prefix, c.year, c.month))
}

func (c crawler) selectionaOrgaoMesAno(ctx context.Context) error {
	const (
		pathRoot = "/html/body/div[2]/input"
		baseURL  = "https://servicos-portal.mpro.mp.br/plcVis/frameset?__report=..%2FROOT%2Frel%2Fcontracheque%2Fmembros%2FremuneracaoMembrosAtivos.rptdesign&anomes="
		finalURL = "&nome=&cargo=&lotacao="
	)
	concatenated := fmt.Sprintf("%s%s%s%s", baseURL, c.year, c.month, finalURL)

	return chromedp.Run(ctx,
		chromedp.Navigate(concatenated),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona caixa de dialogo
		chromedp.Click(`//*[@title='Exportar dados']`, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona as colunas
		chromedp.Click(`/html/body/table/tbody/tr[4]/td[1]/div[1]/div[2]/div/div[1]/table/tbody/tr[5]/td[2]/table/tbody/tr/td/table/tbody/tr[1]/td/input`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
		
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),

		// Aperta no botão de donwload 
		chromedp.Click(`/html/body/table/tbody/tr[4]/td[1]/div[1]/div[2]/div/div[2]/div[2]/div/div[1]/input`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),

		// Altera o diretório de download
		
	)
}

// exportaExcel clica no botão correto para exportar para excel, espera um tempo para download renomeia o arquivo.
func (c crawler) exportaExcel(ctx context.Context, fName string) error {
	err := chromedp.Run(ctx,
		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
	)
	if err != nil {
		return fmt.Errorf("erro clicando no botão de download: %v", err)
	}

	time.Sleep(c.downloadTimeout)

	if err := nomeiaDownload(c.output, fName); err != nil {
		return fmt.Errorf("erro renomeando arquivo (%s): %v", fName, err)
	}
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return fmt.Errorf("download do arquivo de %s não realizado", fName)
	}
	return nil
}

// clicaAba clica na aba referenciada pelo XPATH passado como parâmetro.
// Também espera até o título Tribunal estar visível.
func (c crawler) clicaAba(ctx context.Context, xpath string) error {
	return chromedp.Run(ctx,
		chromedp.Click(xpath),
		chromedp.Sleep(c.timeBetweenSteps),
	)
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