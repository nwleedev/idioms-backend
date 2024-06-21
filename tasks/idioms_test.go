package tasks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"
)

func TestParseResponse(t *testing.T) {
	msg := map[string]interface{}{
		"id":   "OK",
		"body": true,
	}
	text, _ := json.Marshal(msg)
	response := `
	{
		"idiom": "Through thick and thin",
		"meaningBrief": "Steadfast loyalty or support regardless of difficulties or challenges faced.",
		"meaningFull": "The idiom 'Through thick and thin' encapsulates the idea of steadfast loyalty, support, or commitment to someone or something, regardless of the myriad challenges or difficulties that may arise. This expression is commonly employed to characterize enduring bonds—be it friendships, romantic relationships, or alliances—that have withstood the test of time and adversity. Tracing its origins to an old English proverb, the phrase 'through thick and thin' literally referred to navigating dense (thick) or sparse (thin) woodlands, metaphorically paralleling the unpredictable and often arduous journey of life. Utilizing this idiom in conversation not only underscores the depth of one's dedication but also implies a mutual understanding and shared history of overcoming obstacles together. In essence, it speaks to the resilience and unwavering support present in the most meaningful of connections.",
		"examples": [
			"Despite facing numerous setbacks in their business, they stuck together through thick and thin, which eventually led to their success."
		]
	}
	`
	response = fmt.Sprintf("\n```json\n%s\n```\n", string(text))

	promptRe := regexp.MustCompile("```json\n?|```/g")
	content := promptRe.FindStringSubmatch(response)

	t.Log(content)

	if content[0] != string(text) {
		t.Errorf("Expected %s, received %s", string(text), content[0])
		return
	}
	t.Log("No errors found")
}
