package regopls

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/open-policy-agent/opa/misc/regopls/logs"
	"github.com/open-policy-agent/opa/misc/regopls/lsp"
	"github.com/open-policy-agent/opa/misc/regopls/lsp/defines"
)

func strPtr(str string) *string {
	return &str
}

var logPath *string

func init() {
	var logger *log.Logger
	defer func() {
		logs.Init(logger)
	}()
	logPath = flag.String("logs", "", "logs file path")
	if logPath == nil || *logPath == "" {
		logger = log.New(os.Stderr, "", 0)
		return
	}
	p := *logPath
	f, err := os.Open(p)
	if err == nil {
		logger = log.New(f, "", 0)
		return
	}
	f, err = os.Create(p)
	if err == nil {
		logger = log.New(f, "", 0)
		return
	}
	panic(fmt.Sprintf("logs init error: %v", *logPath))
}







func cachedInputFile() map[string]interface{} {
	bytes, _ := ioutil.ReadFile("/Users/davidlu/regopls/rego-test-files/input.json")	// TODO: how does OPA get root?
	var data map[string]interface{}
	json.Unmarshal(bytes, &data)
	return data
}

func StartLanguageServer() {
	documents := make(map[string]string)	// scuffed but allow it for now :)

	server := lsp.NewServer(&lsp.Options{
		Network: "tcp",
		TextDocumentSync: defines.TextDocumentSyncKindFull,
		CompletionProvider: &defines.CompletionOptions{
			TriggerCharacters: &[]string{"."},
		},
	})
	server.OnDidOpenTextDocument(func(ctx context.Context, req *defines.DidOpenTextDocumentParams) error {
		logs.Println("open: ", req)
		documents[string(req.TextDocument.Uri)] = req.TextDocument.Text
		return nil
	})
	server.OnDidChangeTextDocument(func(ctx context.Context, req *defines.DidChangeTextDocumentParams) error {
		logs.Println("change: ", req)
		documents[string(req.TextDocument.Uri)] = req.ContentChanges[0].Text.(string)
		return nil
	})
	server.OnCompletion(func(ctx context.Context, req *defines.CompletionParams) (result *[]defines.CompletionItem, err error) {
		logs.Println("completion: ", req)

		// get token
		lines := strings.Split(documents[string(req.TextDocument.Uri)], "\n")
		line := lines[req.TextDocumentPositionParams.Position.Line]
		logs.Println("line: ", line)
		l := int(req.TextDocumentPositionParams.Position.Character) - 1
		for l >= 0 && l < len(line) {
			if line[l] == ' ' {
				break
			}
			l--
		}
		r := int(req.TextDocumentPositionParams.Position.Character)
		for r < len(line) && r >= 0 {
			if line[r] == ' ' {
				break
			}
			r++
		}
		token := line[l+1:r]
		logs.Println("token: ", token)

		// check if it's input xd
		var items []defines.CompletionItem
		d := defines.CompletionItemKindField
		//x := []defines.CompletionItem{defines.CompletionItem{
		//	Label:      token,
		//	Kind:       &d,
		//	InsertText: strPtr("Hello"),
		//}}

		steps := strings.Split(token, ".")
		if steps[0] == "input" && len(steps) == 2 {	// TODO: can't keep at 2
			json := cachedInputFile()
			prefix := steps[1]	// TODO: can't keep this
			for k, _ := range json {
				if strings.HasPrefix(k, prefix) {
					items = append(items, defines.CompletionItem{
						Label: k,
						Kind: &d,
						InsertText: strPtr(k),
					})
				}
			}


		}

		return &items, nil

	})









	server.OnHover(func(ctx context.Context, req *defines.HoverParams) (result *defines.Hover, err error) {
		logs.Println("hover: ", req)
		return &defines.Hover{Contents: defines.MarkupContent{Kind: defines.MarkupKindPlainText, Value: "Test Hover Hi Derek!"}}, nil
	})
	server.OnDocumentFormatting(func(ctx context.Context, req *defines.DocumentFormattingParams) (result *[]defines.TextEdit, err error) {
		logs.Println("format: ", req)
		line, err := ReadFile(req.TextDocument.Uri)
		if err != nil {
			return nil, err
		}
		res := []defines.TextEdit{}
		for i, v := range line {
			r := convertParagraphs(v)
			if v != r {
				res = append(res, defines.TextEdit{
					Range: defines.Range{
						Start: defines.Position{uint(i), 0},
						End:   defines.Position{uint(i), uint(len(v) + 1)},
					},
					NewText: r,
				})
			}
		}

		return &res, nil
	})
	server.Run()
}

func ReadFile(filename defines.DocumentUri) ([]string, error) {
	enEscapeUrl, _ := url.QueryUnescape(string(filename))
	data, err := ioutil.ReadFile(enEscapeUrl[6:])
	if err != nil {
		return nil, err
	}
	content := string(data)
	line := strings.Split(content, "\n")
	return line, nil
}

// split paragraphs into sentences, and make the sentence first char uppercase and others lowercase
func convertParagraphs(paragraph string) string {
	sentences := []string{}
	for _, sentence := range strings.Split(paragraph, ".") {
		sentence = strings.TrimSpace(sentence)
		s := []string{}
		w := strings.Split(sentence, " ")
		for i, v := range w {
			if len(v) > 0 {
				if i == 0 {
					s = append(s, strings.ToUpper(v[0:1])+strings.ToLower(v[1:]))
				} else {
					s = append(s, strings.ToLower(v))
				}
			}
		}
		if len(s) != 0 {
			sentences = append(sentences, strings.Join(s, " ")+".")
		}
	}
	return strings.Join(sentences, " ")
}
