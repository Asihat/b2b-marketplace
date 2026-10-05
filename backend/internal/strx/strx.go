// Package strx ports the handful of Laravel Str helpers the app relied on.
package strx

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const alphanum = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Random mirrors Str::random: n alphanumeric characters from a CSPRNG.
func Random(n int) string {
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphanum)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		out[i] = alphanum[idx.Int64()]
	}
	return string(out)
}

// UUID returns a random RFC 4122 version 4 UUID.
func UUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

var cyrillic = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "i", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sht", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya", 'ә': "a", 'ғ': "g", 'қ': "k", 'ң': "n", 'ө': "o", 'ұ': "u", 'ү': "u",
	'һ': "h", 'і': "i", 'ї': "i", 'є': "e", 'ґ': "g",
}

// ASCII approximates Str::ascii: strip diacritics, transliterate Cyrillic,
// drop anything that still is not ASCII.
func ASCII(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	stripped, _, err := transform.String(t, s)
	if err != nil {
		stripped = s
	}

	var b strings.Builder
	for _, r := range stripped {
		if r < 128 {
			b.WriteRune(r)
			continue
		}
		lower := unicode.ToLower(r)
		if tr, ok := cyrillic[lower]; ok {
			if unicode.IsUpper(r) && tr != "" {
				tr = strings.ToUpper(tr[:1]) + tr[1:]
			}
			b.WriteString(tr)
			continue
		}
		switch r {
		case 'ß':
			b.WriteString("ss")
		case 'æ', 'Æ':
			b.WriteString("ae")
		case 'ø', 'Ø':
			b.WriteString("o")
		case 'đ', 'Đ':
			b.WriteString("d")
		case 'ł', 'Ł':
			b.WriteString("l")
		case '’', '‘':
			b.WriteString("'")
		case '–', '—':
			b.WriteString("-")
		}
	}
	return b.String()
}

var (
	slugDisallowed = regexp.MustCompile(`[^-a-z0-9\s]+`)
	slugSeparators = regexp.MustCompile(`[-\s]+`)
)

// Slug mirrors Str::slug($title, '-').
func Slug(title string) string {
	s := ASCII(title)
	s = strings.ReplaceAll(s, "@", "-at-")
	s = strings.ToLower(s)
	s = slugDisallowed.ReplaceAllString(s, "")
	s = slugSeparators.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
