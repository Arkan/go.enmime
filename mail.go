package enmime

import (
  "crypto/sha256"
  "encoding/hex"
  "fmt"
  "mime"
  "net/mail"
  "strings"
)

// MIMEBody is the outer wrapper for MIME messages.
type MIMEBody struct {
  Text        string      // The plain text portion of the message
  Html        string      // The HTML portion of the message
  Root        MIMEPart    // The top-level MIMEPart
  Attachments []MIMEPart  // All parts having a Content-Disposition of attachment
  Inlines     []MIMEPart  // All parts having a Content-Disposition of inline
  header      mail.Header // Header from original message
}

// IsMultipartMessage returns true if the message has a recognized multipart Content-Type
// header.  You don't need to check this before calling ParseMIMEBody, it can handle
// non-multipart messages.
func IsMultipart(mediatype string) bool {
  switch mediatype {
  case "multipart/alternative",
    "multipart/mixed",
    "multipart/related",
    "multipart/signed":
    return true
  }

  return false
}

// ParseMIMEBody parses the body of the message object into a  tree of MIMEPart objects,
// each of which is aware of its content type, filename and headers.  If the part was
// encoded in quoted-printable or base64, it is decoded before being stored in the
// MIMEPart object.
func ParseMIMEBody(mailMsg *mail.Message) (*MIMEBody, error) {
  mimeMsg := &MIMEBody{header: mailMsg.Header}
  ctype := mailMsg.Header.Get("Content-Type")
  mediatype, _, _ := mime.ParseMediaType(ctype)

  if !IsMultipart(mediatype) {
    // Mono part
    bodyBytes, err := decodeSection(mailMsg.Header.Get("Content-Transfer-Encoding"),
      ctype, mediatype, mailMsg.Body)
    if err != nil {
      return nil, fmt.Errorf("Error decoding text-only message: %v", err)
    }

    // Check for HTML at top-level, eat errors quietly
    if mediatype == "text/html" {
      mimeMsg.Html = string(bodyBytes)
    } else {
      mimeMsg.Text = string(bodyBytes)
    }
  } else {
    // Parse top-level multipart
    ctype := mailMsg.Header.Get("Content-Type")
    mediatype, params, err := mime.ParseMediaType(ctype)
    if err != nil {
      return nil, fmt.Errorf("Unable to parse media type: %v", err)
    }
    if !strings.HasPrefix(mediatype, "multipart/") {
      return nil, fmt.Errorf("Unknown mediatype: %v", mediatype)
    }
    boundary := params["boundary"]
    if boundary == "" {
      return nil, fmt.Errorf("Unable to locate boundary param in Content-Type header")
    }

    // Root Node of our tree
    root := NewMIMEPart(nil, mediatype)
    mimeMsg.Root = root
    err = parseParts(root, mailMsg.Body, boundary)
    if err != nil {
      return nil, err
    }

    // Locate text body
    if mediatype == "multipart/altern" {
      match := BreadthMatchFirst(root, func(p MIMEPart) bool {
        return p.ContentType() == "text/plain" && p.Disposition() != "attachment"
      })
      if match != nil {
        mimeMsg.Text = string(match.Content())
      }
    } else {
      // multipart is of a mixed type
      match := DepthMatchAll(root, func(p MIMEPart) bool {
        return p.ContentType() == "text/plain" && p.Disposition() != "attachment"
      })
      for i, m := range match {
        if i > 0 {
          mimeMsg.Text += "\n--\n"
        }
        mimeMsg.Text += string(m.Content())
      }
    }

    // Locate HTML body
    match := BreadthMatchFirst(root, func(p MIMEPart) bool {
      return p.ContentType() == "text/html" && p.Disposition() != "attachment"
    })
    if match != nil {
      mimeMsg.Html = string(match.Content())
    }

    // Locate attachments
    mimeMsg.Attachments = BreadthMatchAll(root, func(p MIMEPart) bool {
      // Do not include the parts if they are already present as text or html
      return p.Disposition() == "attachment" && string(p.Content()) != mimeMsg.Html && string(p.Content()) != mimeMsg.Text
    })

    // Locate inlines
    mimeMsg.Inlines = BreadthMatchAll(root, func(p MIMEPart) bool {
      // Do not include the parts if they are already present as text or html
      return p.Disposition() == "inline" && string(p.Content()) != mimeMsg.Html && string(p.Content()) != mimeMsg.Text
    })
  }

  return mimeMsg, nil
}

// Process the specified header for RFC 2047 encoded words and return the result
func (m *MIMEBody) GetHeader(name string) string {
  return decodeHeader(m.header.Get(name))
}

func (m *MIMEBody) generateMessageId() string {
  key := ""
  tld := "revapost.com"
  key += m.GetHeader("FROM")
  key += m.GetHeader("TO")
  key += m.GetHeader("CC")
  key += m.GetHeader("DATE")
  hash := sha256.New()
  hash.Write([]byte(key))
  hashComputed := hash.Sum(nil)
  result := hex.EncodeToString(hashComputed)
  return fmt.Sprintf("<%s-auto-generated@%s>", result, tld)
}

func (m *MIMEBody) MessageId() string {
  if messageId := m.GetHeader("Message-Id"); messageId != "" {
    return messageId
  }
  return m.generateMessageId()
}
