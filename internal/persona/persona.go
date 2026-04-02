package persona

import (
	"encoding/binary"
	"hash/fnv"
	"strings"

	"github.com/disgoorg/disgo/discord"
)

func Mommy(locale discord.Locale) string {
	const (
		mommyDefault = "mommy"

		mommyMummy  = "mummy"
		mommyMama   = "mama"
		mommyMamma  = "mamma"
		mommyMami   = "mami"
		mommyGerman = "Mama"
		mommyFrench = "Maman"
		mommyCzech  = "máma"
		mommyFinn   = "äiti"
		mommyViet   = "mẹ"
		mommyLith   = "mamytė"

		mommySpanishES = "mamá"
		mommyCyrillic  = "мама"

		mommyJapanese = "ママ"
		mommyKorean   = "엄마"
		mommyZHCN     = "妈妈"
		mommyZHTW     = "媽咪"
	)

	// Keep the persona fixed, but allow minor locale-adjacent wording.
	code := strings.ToLower(strings.TrimSpace(locale.Code()))
	switch code {
	case "en-gb":
		return mommyMummy
	case "es-es":
		return mommySpanishES
	case "es-419":
		return mommyMami
	case "de":
		return mommyGerman
	case "fr":
		return mommyFrench
	case "cs":
		return mommyCzech
	case "pl", "nl", "id":
		return mommyMama
	case "it", "sv-se", "no":
		return mommyMamma
	case "fi":
		return mommyFinn
	case "lt":
		return mommyLith
	case "bg", "ru", "uk":
		return mommyCyrillic
	case "vi":
		return mommyViet
	case "ja":
		return mommyJapanese
	case "ko":
		return mommyKorean
	case "zh-cn":
		return mommyZHCN
	case "zh-tw":
		return mommyZHTW
	default:
		return mommyDefault
	}
}

func PetName(locale discord.Locale, userID uint64, messageID string) string {
	terms := petTerms(locale)
	if len(terms) == 0 {
		return "kiddo"
	}

	h := fnv.New64a()
	_, _ = h.Write([]byte(locale.Code()))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.TrimSpace(messageID)))
	_, _ = h.Write([]byte{0})

	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], userID)
	_, _ = h.Write(b[:])

	mod := h.Sum64() % uint64(len(terms))
	maxInt := uint64(^uint(0) >> 1)
	if mod > maxInt {
		mod = 0
	}
	return terms[int(mod)]
}

func petTerms(locale discord.Locale) []string {
	code := strings.ToLower(strings.TrimSpace(locale.Code()))
	if code == "" {
		return defaultPetTerms()
	}

	if terms := petTermsSpanish(code); len(terms) > 0 {
		return terms
	}
	if terms := petTermsEurope(code); len(terms) > 0 {
		return terms
	}
	if terms := petTermsCyrillic(code); len(terms) > 0 {
		return terms
	}
	if terms := petTermsAsia(code); len(terms) > 0 {
		return terms
	}

	return defaultPetTerms()
}

func defaultPetTerms() []string {
	return []string{"kiddo", "sweetie", "little one", "honey", "pumpkin", "sunshine", "love", "dear"}
}

func petTermsSpanish(code string) []string {
	switch code {
	case "es-es":
		return []string{"cariño", "cielo", "tesoro", "corazón", "peque", "amor"}
	case "es-419":
		return []string{"cariño", "cielito", "tesoro", "corazón", "chiqui", "mi amor"}
	default:
		return nil
	}
}

func petTermsEurope(code string) []string {
	switch code {
	case "da":
		return []string{"skat", "søde", "lille ven"}
	case "de":
		return []string{"liebling", "schatz", "kleines"}
	case "fi":
		return []string{"kulta", "rakas", "pikkuinen", "muru", "sydänkäpynen"}
	case "fr":
		return []string{"mon cœur", "petit cœur", "mon trésor"}
	case "hr":
		return []string{"srce", "dušo", "mali"}
	case "hu":
		return []string{"drágám", "kicsim", "édesem"}
	case "it":
		return []string{"tesoro", "cuoricino", "piccolino"}
	case "lt":
		return []string{"mielas", "širdelė", "mažyli"}
	case "nl":
		return []string{"lieverd", "schat", "kleintje"}
	case "no":
		return []string{"kjære", "skatt", "lille venn"}
	case "pl":
		return []string{"kochanie", "skarbie", "słonko"}
	case "pt-br":
		return []string{"querido", "docinho", "meu bem"}
	case "ro":
		return []string{"pui", "suflet", "drag"}
	case "sv-se":
		return []string{"älskling", "vännen", "lilla vän"}
	case "cs":
		return []string{"zlatíčko", "sluníčko", "drobku"}
	case "el":
		return []string{"καρδιά μου", "αγάπη μου", "μικρό μου"}
	case "tr":
		return []string{"canım", "tatlım", "küçüğüm"}
	default:
		return nil
	}
}

func petTermsCyrillic(code string) []string {
	switch code {
	case "bg":
		return []string{"слънчице", "сърчице", "миличко"}
	case "ru":
		return []string{"солнышко", "зайка", "котик"}
	case "uk":
		return []string{"сонечко", "зайченя", "котик"}
	default:
		return nil
	}
}

func petTermsAsia(code string) []string {
	switch code {
	case "id":
		return []string{"sayang", "manis", "teman kecil"}
	case "hi":
		return []string{"बच्चे", "प्यारे", "नन्हे"}
	case "th":
		return []string{"ที่รัก", "คนเก่ง", "เด็กน้อย"}
	case "vi":
		return []string{"bé", "cưng", "nhỏ"}
	case "zh-cn":
		return []string{"宝贝", "小可爱", "乖乖"}
	case "zh-tw":
		return []string{"寶貝", "小可愛", "乖乖"}
	case "ja":
		return []string{"いい子", "ハニー", "かわいい子"}
	case "ko":
		return []string{"아가", "꼬마야", "귀염둥이"}
	default:
		return nil
	}
}
