package html

// VoidTags https://html.spec.whatwg.org/multipage/syntax.html#void-elements
var VoidTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true, "command": true,
	"keygen": true, "menuitem": true,
}
