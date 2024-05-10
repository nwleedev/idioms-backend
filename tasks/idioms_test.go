package tasks

import (
	"encoding/json"
	"regexp"
	"testing"
)

func TestParseResponse(t *testing.T) {
	msg := map[string]interface{}{
		"id":   "OK",
		"body": true,
	}
	text, _ := json.Marshal(msg)
	response := "```json\n{\n  \"idiom\": \"Hit the hay\",\n  \"meaningBrief\": \"To go to bed or start sleeping.\",\n  \"meaningFull\": \"The phrase 'hit the hay' originally comes from a time when mattresses were often filled with hay or straw. To 'hit the hay' meant to literally lie down on one's hay-filled mattress, thus signifying going to bed or falling asleep. Over time, the saying evolved beyond its agricultural roots to become a common colloquialism used by people everywhere, regardless of whether their beds have any association with hay. Today, it simply means to go to bed or to go to sleep, and it's used in casual conversation.\",\n  \"description\": \"After a long day of hiking and exploring the scenic trails, the exhausted group decided it was time to hit the hay early to rest up for another adventurous day ahead.\",\n  \"examples\": [\n    \"I've got to wake up early for work tomorrow, so I think I'll hit the hay now.\",\n    \"After the movie marathon, we were all so tired that we decided to hit the hay.\",\n    \"She hit the hay right after dinner, completely worn out from the day's events.\",\n    \"The kids played all day in the yard. By sunset, they were ready to hit the hay.\",\n    \"He studied until midnight for his exams, then hit the hay to get a few hours of rest.\",\n    \"We've got a big day ahead of us tomorrow, so let's hit the hay and get a good night's sleep.\",\n    \"After the long road trip, hitting the hay was the first thing on everyone's mind.\",\n    \"She hit the hay early to be well-rested for her job interview in the morning.\",\n    \"The party was fun, but around 2 AM, I had to hit the hay since I couldn't keep my eyes open.\",\n    \"After finishing the book he'd been reading for hours, he finally decided to hit the hay.\"\n  ]\n}\n```"
	// esponse := fmt.Sprintf("```json%s```", string(text))

	promptRe := regexp.MustCompile("```json(.*)```")
	content := promptRe.FindStringSubmatch(response)

	t.Log(content)

	if content[0] != string(text) {
		t.Errorf("Expected %s, received %s", string(text), content[0])
		return
	}
	t.Log("No errors found")
}
