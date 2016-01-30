package cleaner

import (
  "golang.org/x/net/html"
  "golang.org/x/net/html/atom"
  "strings"
  "bytes"
)

// finds article's title and body in ria.ru html style
// works cleary on 15.12.2015
func FindTitleAndBody_Ria(node *html.Node) (*html.Node, *html.Node) {
    var title, fulltext *html.Node

    if node.Type == html.ElementNode {
        for _, tag := range node.Attr {
          if tag.Key == "itemprop" {
            if tag.Val == "articleBody" {
              node.Data = "body"
              fulltext = node
              break
            }
            if tag.Val == "name" {
              node.Data = "title"
              title = node
              break
            }
          }
        }
    }

  for c:= node.FirstChild; c != nil; c = c.NextSibling {
    ptitle, pfulltext := FindTitleAndBody_Ria(c)
    if ptitle != nil {
        title = ptitle
    }
    if pfulltext != nil {
        fulltext = pfulltext
    }
    if title != nil && fulltext != nil {
        break
    }
  }
  return title, fulltext

}

func FindTitleMK(node *html.Node) (*html.Node)  {
    var title *html.Node
    if node.Type == html.ElementNode {
        if node.Data == "h1" {
            return node
        }
    }
    for c:= node.FirstChild; c != nil; c = c.NextSibling {
        title = FindTitleMK(c)
        if title != nil {
            break
        }
    }
    return title
}

func FindTitleAndBody_MK(node *html.Node) (*html.Node, *html.Node) {
    var title, fulltext *html.Node

    if node.Type == html.ElementNode {
        for _, tag := range node.Attr {
            if tag.Key == "class" {
                if tag.Val == "content" {
                    title = FindTitleMK(node)
                    node.Data = "body"
                    fulltext = node
                    break
                }
            }
        }
    }

  for c:= node.FirstChild; c != nil; c = c.NextSibling {
      ptitle, pfulltext := FindTitleAndBody_MK(c)
      if ptitle != nil {
          title = ptitle
          title.Data = "title"
      }
      if pfulltext != nil {
          fulltext = pfulltext
      }
      if title != nil && fulltext != nil {
          break
      }
  }
  return title, fulltext

}

//return true if need to delete node, false another way
func deleteValuelessNodes(innode *html.Node) (bool)  {
    if innode.Type == html.CommentNode {
        //fmt.Println("comment:" + innode.Data)
        return true
    }
    if innode.Type == html.ElementNode {
        //innode.Attr = []html.Attribute{}
        if innode.Data == "script" || innode.Data == "meta" || innode.Data == "style" || innode.Data == "head" || innode.Data == "form" || innode.Data == "noscript" || innode.Data == "img" || innode.Data == "noindex" || innode.Data == "span" {
            //fmt.Println("script: " + innode.Data)
            return true
        }
    }
    for node:= innode.FirstChild; node != nil; {
        if deleteValuelessNodes(node) {
            tnode := node.NextSibling
            innode.RemoveChild(node)
            node = tnode
            continue
        }
        node = node.NextSibling
    }
    return false
}

func makeHtml(title_n *html.Node, body_n *html.Node) (*html.Node)  {
    if title_n == nil {
        // make manually title node
        title_n = new(html.Node)
        *title_n = html.Node{
             Parent: nil,
             FirstChild: nil,
             LastChild: nil,
             PrevSibling: nil,
             NextSibling: nil,
             Type: html.ElementNode,
             DataAtom: atom.Title,
             Data: "title",
             Attr: []html.Attribute{},
         }
         title_text := new(html.Node)
         *title_text = html.Node{
              Parent: nil,
              FirstChild: nil,
              LastChild: nil,
              PrevSibling: nil,
              NextSibling: nil,
              Type: html.TextNode,
              DataAtom: 0,
              Data: "Empty title",
              Attr: []html.Attribute{},
          }
          title_n.AppendChild(title_text)
    } else {
        // clear tag from parametrs
        title_n.Attr = []html.Attribute{}
        // remove parents for correct work
        title_n.Parent.RemoveChild(title_n)
    }
    if body_n == nil {
        // make manually body node
        body_n = new(html.Node)
        *body_n = html.Node{
             Parent: nil,
             FirstChild: nil,
             LastChild: nil,
             PrevSibling: nil,
             NextSibling: nil,
             Type: html.ElementNode,
             DataAtom: atom.Body,
             Data: "body",
             Attr: []html.Attribute{},
         }
         body_text := new(html.Node)
         *body_text = html.Node{
              Parent: nil,
              FirstChild: nil,
              LastChild: nil,
              PrevSibling: nil,
              NextSibling: nil,
              Type: html.TextNode,
              DataAtom: 0,
              Data: "Empty body",
              Attr: []html.Attribute{},
          }
          body_n.AppendChild(body_text)
    } else {
        body_n.Attr = []html.Attribute{}
        body_n.Parent.RemoveChild(body_n)
    }
    model := "<html><head><meta charset=\"utf-8\"></head></html>"
    output, _ := html.Parse(strings.NewReader(model))

    htmlnode := output.FirstChild
    headnode := htmlnode.FirstChild
    defbodynode := headnode.NextSibling
    output.FirstChild.RemoveChild(defbodynode) // delete empty <body> tag
    headnode.AppendChild(title_n)
    htmlnode.AppendChild(body_n)

    return output
}

// clear doc of tags and not target text
// works cleary on 30.01.2016 with current(today) ria.ru html style
// in future may not work
func ClearRIA(input []byte) ([]byte, bool)  {
    pureString := string(input)
    doc, err := html.Parse(strings.NewReader(pureString))
    if err != nil {
        return nil, false
    }
    deleteValuelessNodes(doc)
    title, body := FindTitleAndBody_Ria(doc)
    output := makeHtml(title, body)
    outbyte:= new(bytes.Buffer)
    err = html.Render(outbyte, output)
    if err != nil {
        return nil, false
    }
    return outbyte.Bytes(), true
}

// clear doc of tags and not target text
// works cleary on 30.01.2016 with current(today) www.mk.ru html style
// in future may not work
func ClearMK(input []byte) ([]byte, bool)  {
    pureString := string(input)
    doc, err := html.Parse(strings.NewReader(pureString))
    if err != nil {
        return nil, false
    }
    deleteValuelessNodes(doc)
    title, body := FindTitleAndBody_MK(doc)
    output := makeHtml(title, body)
    outbyte:= new(bytes.Buffer)
    err = html.Render(outbyte, output)
    if err != nil {
        return nil, false
    }
    return outbyte.Bytes(), true
}
