package main

import (
 "bytes"
 "errors"
 "regexp"
 "strings"

 xhtml "golang.org/x/net/html"
 "golang.org/x/net/html/atom"
)

var notesColor = regexp.MustCompile(`(?i)^(#[0-9a-f]{3}|#[0-9a-f]{6}|black|white|red|green|blue|gray|grey|yellow|orange|purple|transparent|rgb\(\s*[0-9]{1,3}\s*,\s*[0-9]{1,3}\s*,\s*[0-9]{1,3}\s*\))$`)
var notesFonts = map[string]bool{"arial":true,"georgia":true,"times new roman":true,"verdana":true,"trebuchet ms":true,"courier new":true,"sans-serif":true,"serif":true,"monospace":true}
var notesTags = map[string]bool{"div":true,"p":true,"br":true,"ul":true,"ol":true,"li":true,"span":true,"b":true,"strong":true,"i":true,"em":true,"u":true,"font":true}
var notesDrop = map[string]bool{"script":true,"style":true,"iframe":true,"object":true,"embed":true,"svg":true,"math":true,"template":true,"noscript":true,"head":true,"title":true,"link":true,"meta":true,"base":true}

func safeNotesStyle(name, value string) bool {
 switch name {
 case "color": return notesColor.MatchString(value)
 case "font-family": return notesFonts[strings.ToLower(strings.Trim(value, "\"'"))]
 case "font-weight": return value == "normal" || value == "bold" || value == "400" || value == "700"
 case "font-style": return value == "normal" || value == "italic"
 case "text-decoration": return value == "none" || value == "underline" || value == "line-through"
 }
 return false
}

// Build new nodes; never insert untrusted parser nodes or attributes into output.
func sanitizeNotesHTML(input string) (string, error) {
 if len(input) > 2*1024*1024 { return "", errors.New("formatted notes exceed the 2 MiB limit") }
 context := &xhtml.Node{Type:xhtml.ElementNode,Data:"div",DataAtom:atom.Div}
 nodes, err := xhtml.ParseFragment(strings.NewReader(input), context)
 if err != nil { return "", err }
 out := &xhtml.Node{Type:xhtml.ElementNode,Data:"div",DataAtom:atom.Div}
 var copyNode func(*xhtml.Node,*xhtml.Node,int) error
 copyNode = func(src,dst *xhtml.Node, depth int) error {
  if depth > 128 { return errors.New("formatted notes nesting is too deep") }
  if src.Type == xhtml.TextNode { dst.AppendChild(&xhtml.Node{Type:xhtml.TextNode,Data:src.Data}); return nil }
  if src.Type != xhtml.ElementNode || src.Namespace != "" || notesDrop[src.Data] { return nil }
  target := dst
  if notesTags[src.Data] {
   target = &xhtml.Node{Type:xhtml.ElementNode,Data:src.Data,DataAtom:src.DataAtom}
   for _, attr := range src.Attr {
    if attr.Namespace != "" { continue }
    if attr.Key == "style" {
     safe := []string{}
     for _, declaration := range strings.Split(attr.Val,";") {
      pair := strings.SplitN(declaration,":",2)
      if len(pair) != 2 { continue }
      name,value := strings.ToLower(strings.TrimSpace(pair[0])),strings.TrimSpace(pair[1])
      if safeNotesStyle(name,value) { safe = append(safe,name+": "+value) }
     }
     if len(safe)>0 { target.Attr=append(target.Attr,xhtml.Attribute{Key:"style",Val:strings.Join(safe,"; ")}) }
    } else if src.Data=="font" && ((attr.Key=="color" && safeNotesStyle("color",attr.Val)) || (attr.Key=="face" && safeNotesStyle("font-family",attr.Val))) {
     target.Attr=append(target.Attr,xhtml.Attribute{Key:attr.Key,Val:attr.Val})
    }
   }
   dst.AppendChild(target)
  }
  for child:=src.FirstChild;child!=nil;child=child.NextSibling { if err:=copyNode(child,target,depth+1);err!=nil{return err} }
  return nil
 }
 for _, node := range nodes { if err:=copyNode(node,out,0);err!=nil{return "",err} }
 var result bytes.Buffer
 for child:=out.FirstChild;child!=nil;child=child.NextSibling { if err:=xhtml.Render(&result,child);err!=nil{return "",err} }
 return result.String(),nil
}
