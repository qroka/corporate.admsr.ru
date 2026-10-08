// Package profanity — поиск мата и грубой брани в пользовательском тексте
// (комментарии к новостям, записи на стене профиля; ADR-053).
//
// Слово нормализуется в кириллицу несколькими способами, потому что одна и та же
// латинская буква значит разное: в транслите «p» — это «п» (pizda), а как похожая
// буква — «р»; «3» — это «з» (пи3да) или «е» (3блан). Затем схлопываются повторы
// (хууууй → хуй) и проверяются корни. Слова, разбитые точками, дефисами или
// пробелами по одной букве (х.у.й, х у й), склеиваются.
//
// Корни, которые встречаются внутри обычных слов («еб» — хлеб, небо, вебинар;
// «бля» — употреблять, корабля; «манд» — команда), ищутся только в начале слова,
// с приставкой или целой словоформой. Список — только здесь, на фронте его нет.
package profanity

import (
	"regexp"
	"strings"
	"unicode"
)

// Span — найденное слово: позиции его букв в исходном тексте (индексы рун).
// Знаки между буквами (х.у.й) в Positions не входят.
type Span struct {
	Positions []int
}

// Check — есть ли в тексте мат и текст, где в найденных словах буквы между
// первой и последней заменены на «*».
func Check(text string) (masked string, found bool) {
	spans := Find(text)
	if len(spans) == 0 {
		return text, false
	}
	return Mask(text, spans), true
}

// Mask — в каждом найденном слове оставить первую и последнюю букву, остальные — «*».
func Mask(text string, spans []Span) string {
	runes := []rune(text)
	for _, s := range spans {
		p := s.Positions
		switch n := len(p); {
		case n >= 3:
			for _, i := range p[1 : n-1] {
				runes[i] = '*'
			}
		case n == 2:
			runes[p[1]] = '*'
		case n == 1:
			runes[p[0]] = '*'
		}
	}
	return string(runes)
}

// Find — все слова с матом в тексте.
func Find(text string) []Span {
	runes := []rune(text)
	var spans []Span

	// Куски без пробелов; внутри — части из «буквенных» знаков, разделённые прочими (х.у.й, по-ебански).
	type chunk struct{ parts [][]int }
	var chunks []chunk
	for i := 0; i < len(runes); {
		if unicode.IsSpace(runes[i]) {
			i++
			continue
		}
		var c chunk
		var part []int
		for ; i < len(runes) && !unicode.IsSpace(runes[i]); i++ {
			if isWordRune(runes[i]) {
				part = append(part, i)
				continue
			}
			if len(part) > 0 {
				c.parts = append(c.parts, part)
				part = nil
			}
		}
		if len(part) > 0 {
			c.parts = append(c.parts, part)
		}
		if len(c.parts) > 0 {
			chunks = append(chunks, c)
		}
	}

	for _, c := range chunks {
		hit := false
		for _, p := range c.parts {
			if isProfane(runes, p) {
				spans = append(spans, Span{Positions: p})
				hit = true
			}
		}
		if !hit && len(c.parts) > 1 {
			var joined []int
			for _, p := range c.parts {
				joined = append(joined, p...)
			}
			if isProfane(runes, joined) {
				spans = append(spans, Span{Positions: joined})
			}
		}
	}

	// По одной букве через пробел: «х у й», «е б а т ь» — от трёх кусков подряд.
	for i := 0; i < len(chunks); {
		j := i
		var joined []int
		for j < len(chunks) && len(chunks[j].parts) == 1 && len(chunks[j].parts[0]) == 1 {
			joined = append(joined, chunks[j].parts[0][0])
			j++
		}
		if j-i >= 3 && isProfane(runes, joined) {
			spans = append(spans, Span{Positions: joined})
		}
		if j == i {
			j++
		}
		i = j
	}
	return spans
}

// isWordRune — буква, цифра или символ, которым подменяют букву (@ → а, $ → с).
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '@' || r == '$' || r == '€'
}

func isProfane(runes []rune, positions []int) bool {
	word := make([]rune, len(positions))
	letters, digits := 0, 0
	for i, p := range positions {
		r := unicode.ToLower(runes[p])
		word[i] = r
		if unicode.IsLetter(r) {
			letters++
		} else {
			digits++
		}
	}
	// «36а», «2026» — номера, а не слова с подменой букв.
	if letters == 0 || digits > letters {
		return false
	}
	if latinWhitelist[string(word)] {
		return false
	}
	for _, v := range variants(word) {
		v = exceptionsRe.ReplaceAllString(v, "#")
		for _, re := range rootPatterns {
			if re.MatchString(v) {
				return true
			}
		}
	}
	return false
}

// variants — кириллические прочтения слова: транслит / похожие буквы × «3» как «з» / «е».
func variants(word []rune) []string {
	threes := []rune{'з'}
	for _, r := range word {
		if r == '3' {
			threes = append(threes, 'е')
			break
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, lookalike := range []bool{false, true} {
		for _, three := range threes {
			v := toCyrillic(word, lookalike, three)
			// И без схлопывания: в «переебать» двойное «е» — приставка + корень.
			for _, s := range []string{collapse(v), v} {
				if !seen[s] {
					seen[s] = true
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// Транслит в несколько латинских букв — только при чтении «транслитом».
var digraphs = []struct {
	lat string
	cyr string
}{
	{"sch", "щ"}, {"sh", "ш"}, {"ch", "ч"}, {"zh", "ж"}, {"kh", "х"}, {"ts", "ц"},
	{"ya", "я"}, {"ja", "я"}, {"yo", "йо"}, {"jo", "йо"}, {"yu", "ю"}, {"ju", "ю"}, {"ye", "е"},
}

// Латиница как транслит: pizda, hui, blyad.
var translit = map[rune]rune{
	'a': 'а', 'b': 'б', 'c': 'ц', 'd': 'д', 'e': 'е', 'f': 'ф', 'g': 'г', 'h': 'х', 'i': 'и',
	'j': 'й', 'k': 'к', 'l': 'л', 'm': 'м', 'n': 'н', 'o': 'о', 'p': 'п', 'q': 'к', 'r': 'р',
	's': 'с', 't': 'т', 'u': 'у', 'v': 'в', 'w': 'в', 'x': 'х', 'y': 'й', 'z': 'з',
}

// Латиница как похожие буквы: xyй, cyka, nизда. Чего здесь нет — берётся из транслита.
var lookalikes = map[rune]rune{
	'a': 'а', 'b': 'ь', 'c': 'с', 'e': 'е', 'h': 'н', 'k': 'к', 'm': 'м', 'n': 'п', 'o': 'о',
	'p': 'р', 'r': 'г', 't': 'т', 'u': 'и', 'x': 'х', 'y': 'у',
}

// Цифры и символы вместо букв: е6ать, 0, бл9, @. «3» — отдельно (з или е).
var symbols = map[rune]rune{
	'0': 'о', '1': 'и', '4': 'ч', '6': 'б', '8': 'в', '9': 'я', '@': 'а', '$': 'с', '€': 'е',
}

func toCyrillic(word []rune, lookalike bool, three rune) string {
	var b strings.Builder
	for i := 0; i < len(word); i++ {
		r := word[i]
		if !lookalike && r < unicode.MaxASCII {
			if cyr, n := matchDigraph(word[i:]); n > 0 {
				b.WriteString(cyr)
				i += n - 1
				continue
			}
		}
		switch {
		case r == 'ё':
			b.WriteRune('е')
		case r == '3':
			b.WriteRune(three)
		case symbols[r] != 0:
			b.WriteRune(symbols[r])
		case lookalike && lookalikes[r] != 0:
			b.WriteRune(lookalikes[r])
		case translit[r] != 0:
			b.WriteRune(translit[r])
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func matchDigraph(rest []rune) (string, int) {
	for _, d := range digraphs {
		n := len(d.lat)
		if len(rest) >= n && string(rest[:n]) == d.lat {
			return d.cyr, n
		}
	}
	return "", 0
}

// collapse — «хууууй» → «хуй», «ссука» → «сука».
func collapse(s string) string {
	var b strings.Builder
	var prev rune = -1
	for _, r := range s {
		if r != prev {
			b.WriteRune(r)
		}
		prev = r
	}
	return b.String()
}

// Английские слова, которые иначе читаются как мат (eBay → «ебай»).
var latinWhitelist = map[string]bool{
	"ebay": true, "ebook": true, "ebooks": true, "ebola": true, "ebit": true, "ebitda": true,
	"hue": true, "ibanez": true,
}

// Обычные слова с «матным» корнем внутри — вырезаются до проверки корней.
var exceptionsRe = regexp.MustCompile(`страху|скипидар|^ебург|^ебитда|^ибанез`)

// Приставки перед «еб»: заебал, наебать, отъебись, долбоёб. «в», «с» — только с «ъ»
// (въебать, съебаться), иначе вебинар, себе.
const ebPrefix = `(?:за|на|вы|по|у|до|про|при|пере|недо|не|ра[зс]ъ?|отъ?|объ?|подъ?|надъ?|изъ?|исъ?|взъ?|въ|съ|долб[оа]|мозго|мудо)?`

var rootPatterns = func() []*regexp.Regexp {
	src := []string{
		// хуй, хуёвый, хуеплёт, нахуя, похую, охуеть
		`ху[йеиюя]`,
		`^хули$`,
		// пизда, пиздец, спиздил, пезда
		`п[ие]зд|пизж`,
		// ебать, ёбаный, заебал, уёбок, ебло, еблан, долбоёб; йобаный (но не job)
		`^` + ebPrefix + `(?:еб(?:[аеиоуыюялнш]|$)|йоб[аеиоуын])`,
		// ибать, заибал
		`^(?:за|на|вы|по|у|до|про|отъ?|ра[зс]ъ?|подъ?)иб(?:а[лнт]|ну)|^иба[лтн]`,
		// блядь, блять, бля; выблядок
		`бляд|^(?:вы)?бля(?:т|$)`,
		// манда — только словоформы (команда, мандарин, мандат — нет)
		`^манд(?:а|ы|е|у|ой|ою|ец|ища|авош.*)$`,
		`залуп`,
		`муд(?:ак|ач|ил|озв)`,
		// сука — только словоформы (сукно, суккуб — нет)
		`^сук(?:а|и|е|у|ой|ою|ин|ина|ины|ам|ами|ах)$`,
		`^суч(?:ар|онок|ат)`,
		`пид[оа]р|пидр|педераст`,
		`^педик(?:и|а|ов|у|ом)?$`,
		`г[ао]нд[оа]н`,
		`шлюх|шлюш`,
		`г[ао]вн`,
		`мраз[ьиоеу]`,
		`^чмо(?:$|ш)`,
		`^(?:за|по|на|от|под|до)?дроч`,
	}
	out := make([]*regexp.Regexp, len(src))
	for i, s := range src {
		out[i] = regexp.MustCompile(s)
	}
	return out
}()
