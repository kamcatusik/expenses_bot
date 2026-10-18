// Package logx защищает логи от утечки секретов.
package logx

import (
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

// Токен Telegram встречается в URL запроса (…/bot<id>:<secret>/method), а Go вставляет URL в текст сетевых ошибок.
var telegramToken = regexp.MustCompile(`bot\d+:[A-Za-z0-9_-]{20,}`)

type redactor struct {
	w       io.Writer
	secrets []string
}

func (r *redactor) Write(p []byte) (int, error) {
	s := string(p)
	for _, sec := range r.secrets {
		s = strings.ReplaceAll(s, sec, "***")
	}
	s = telegramToken.ReplaceAllString(s, "bot***")
	if _, err := r.w.Write([]byte(s)); err != nil {
		return 0, err
	}
	return len(p), nil // вызывающий log ждёт длину исходных данных
}

// Setup заменяет вывод стандартного логгера: каждое значение из secrets
// (и любой токен Telegram по маске) затирается в каждой строке лога.
func Setup(secrets ...string) {
	var list []string
	for _, s := range secrets {
		if len(s) >= 8 { // пустые и короткие значения не трогаем, иначе испортим весь лог
			list = append(list, s)
		}
	}
	log.SetOutput(&redactor{w: os.Stderr, secrets: list})
}
