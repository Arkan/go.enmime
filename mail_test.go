package enmime

import (
  "bufio"
  "bytes"
  "fmt"
  "net/mail"
  "os"
  "path/filepath"
  "testing"
  "mime"

  "github.com/stretchr/testify/assert"
)

func TestIdentifySinglePart(t *testing.T) {
  msg := readMessage("non-mime.raw")
  ctype := msg.Header.Get("Content-Type")
  mediatype, _, _ := mime.ParseMediaType(ctype)
  assert.False(t, IsMultipart(mediatype), "Failed to identify non-multipart message")
}

func TestIdentifyMultiPart(t *testing.T) {
  msg := readMessage("html-mime-inline.raw")
  ctype := msg.Header.Get("Content-Type")
  mediatype, _, _ := mime.ParseMediaType(ctype)
  assert.True(t, IsMultipart(mediatype), "Failed to identify multipart MIME message")
}

func TestParseNonMime(t *testing.T) {
  msg := readMessage("non-mime.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }

  assert.Contains(t, mime.Text, "This is a test mailing")
  assert.Empty(t, mime.Html, "Expected no HTML body")
}

func TestParseNonMimeHtml(t *testing.T) {
  msg := readMessage("non-mime-html.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }

  assert.Contains(t, mime.Text, "")
  assert.Contains(t, mime.Html, "This is a test mailing")
}

func TestParseMimeTree(t *testing.T) {
  msg := readMessage("attachment.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.NotNil(t, mime.Root, "Message should have a root node")
}

func TestParseInlineText(t *testing.T) {
  msg := readMessage("html-mime-inline.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Equal(t, "Test of text section", mime.Text)
}

func TestParseMultiMixedText(t *testing.T) {
  msg := readMessage("mime-mixed.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Equal(t, mime.Text, "Section one\n\n--\nSection two",
    "Text parts should be concatenated")
}

func TestParseMultiSignedText(t *testing.T) {
  msg := readMessage("mime-signed.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Equal(t, mime.Text, "Section one\n\n--\nSection two",
    "Text parts should be concatenated")
}

func TestParseQuotedPrintable(t *testing.T) {
  msg := readMessage("quoted-printable.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Contains(t, mime.Text, "Phasellus sit amet arcu")
}

func TestParseQuotedPrintableMime(t *testing.T) {
  msg := readMessage("quoted-printable-mime.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Contains(t, mime.Text, "Nullam venenatis ante")
}

func TestParseInlineHtml(t *testing.T) {
  msg := readMessage("html-mime-inline.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Contains(t, mime.Html, "<html>")
  assert.Contains(t, mime.Html, "Test of HTML section")
}

func TestParseAttachment(t *testing.T) {
  msg := readMessage("attachment.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Contains(t, mime.Text, "A text section")
  assert.Equal(t, "", mime.Html, "Html attachment is not for display")
  assert.Equal(t, 0, len(mime.Inlines), "Should have no inlines")
  assert.Equal(t, 1, len(mime.Attachments), "Should have a single attachment")
  assert.Equal(t, "test.html", mime.Attachments[0].FileName(), "Attachment should have correct filename")
  assert.Contains(t, string(mime.Attachments[0].Content()), "<html>",
    "Attachment should have correct content")

  //for _, a := range mime.Attachments {
  //	fmt.Printf("%v %v\n", a.ContentType(), a.Disposition())
  //}
}

func TestParseInline(t *testing.T) {
  msg := readMessage("html-mime-inline.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Contains(t, mime.Text, "Test of text section", "Should have text section")
  assert.Contains(t, mime.Html, ">Test of HTML section<", "Should have html section")
  assert.Equal(t, 1, len(mime.Inlines), "Should have one inline")
  assert.Equal(t, 0, len(mime.Attachments), "Should have no attachments")
  assert.Equal(t, "favicon.png", mime.Inlines[0].FileName(), "Inline should have correct filename")
  assert.True(t, bytes.HasPrefix(mime.Inlines[0].Content(), []byte{0x89, 'P', 'N', 'G'}),
    "Content should be PNG image")

}

func TestParseNestedHeaders(t *testing.T) {
  msg := readMessage("html-mime-inline.raw")
  mime, err := ParseMIMEBody(msg)

  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Equal(t, 1, len(mime.Inlines), "Should have one inline")
  assert.Equal(t, "favicon.png", mime.Inlines[0].FileName(), "Inline should have correct filename")
  assert.Equal(t, "<8B8481A2-25CA-4886-9B5A-8EB9115DD064@skynet>", mime.Inlines[0].Header().Get("Content-Id"), "Inline should have a Content-Id header")
}

func TestParseEncodedSubject(t *testing.T) {
  // Even non-MIME messages should support encoded-words in headers
  msg := readMessage("qp-ascii-header.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  assert.Equal(t, "Test QP Subject!", mime.GetHeader("Subject"))

  // Test UTF-8 subject line
  msg = readMessage("qp-utf8-header.raw")
  mime, err = ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }
  assert.Equal(t, "MIME UTF8 Test \u00a2 More Text", mime.GetHeader("Subject"))
}

func TestSparrow(t *testing.T) {
  msg := readMessage("sparrow.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  assert.Equal(t, "LOLZ3", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum\n\n--\nJames\n\n", mime.Text)
  assert.Equal(t, "                <div>\n                    <span style=\"font-family: Arial, Helvetica, sans; font-size: 11px; line-height: 14px; text-align: justify;\">Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum</span>\n                </div>\n                <div><div><br></div><div>--&nbsp;</div><div>Florian Bertholin</div><div><br></div></div>\n", mime.Html)
}

// func Test01(t *testing.T) {
//   msg := readMessage("01-body_is_an_attachment.eml")
//   mime, err := ParseMIMEBody(msg)
//   if err != nil {
//     t.Fatalf("Failed to parse non-MIME: %v", err)
//   }
//   _ = mime
//   assert.Equal(t, "Test photo", mime.GetHeader("Subject"))
//   assert.Equal(t, 0, len(mime.Attachments))
//   assert.Equal(t, 0, len(mime.Inlines))
//   assert.Equal(t, "", mime.Text)
//   assert.Equal(t, "", mime.Html)
// }

func Test02(t *testing.T) {
  msg := readMessage("02-inline_pictures.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "test text parts", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 2, len(mime.Inlines))
  assert.Equal(t, "youpi\r\n\r\n\r\n\r\n\r\ntrop bi1\r\n\r\n\r\n\r\n\r\nburp\r\n\r\n", mime.Text)
  assert.Equal(t, "<html><head><meta http-equiv=\"Content-Type\" content=\"text/html charset=us-ascii\"></head><body style=\"word-wrap: break-word; -webkit-nbsp-mode: space; -webkit-line-break: after-white-space;\">youpi<div><br></div><div><img height=\"250\" width=\"210\" apple-width=\"yes\" apple-height=\"yes\" apple-inline=\"yes\" id=\"F4FA4AF9-956C-43CE-9E03-2CA6D2A59952\" src=\"cid:F2E213E4-D044-4BEF-982D-12FCB8701F31@hillyerd.com\"></div><div><br></div><div><br></div><div>trop bi1</div><div><br></div><div><img height=\"235\" width=\"250\" apple-width=\"yes\" apple-height=\"yes\" apple-inline=\"yes\" id=\"997EE7FA-0B89-4977-9D86-9117B1295E10\" src=\"cid:E438E104-EC3A-4AF5-B985-2148A44E48AE@hillyerd.com\"></div><div><br></div><div><br></div><div>burp</div><div><br></div></body></html>", mime.Html)
}

func Test03(t *testing.T) {
  msg := readMessage("03-monopart_html_only.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Test mail monopart HTML only", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "", mime.Text)
  assert.Equal(t, "<html><head>\r\n<meta content=\"text/html; charset=ISO-8859-1\" http-equiv=\"Content-Type\">\r\n</head><body bgcolor=\"#FFFFFF\" text=\"#000000\">\r\nCeci est une part HTML. Il n'y a pas de part TEXT<br>\r\n  <br>\r\n  <span style=\"font-weight: bold;\">YOUPIIII<br>\r\n    <br>\r\n  </span>Voila.<span style=\"font-weight: bold;\"><span\r\nstyle=\"font-weight: bold;\"><br>\r\n      <br>\r\n    </span></span>\r\n</body>\r\n</html>\r\n\r\n", mime.Html)
}

func Test04(t *testing.T) {
  msg := readMessage("04-monopart_plain_text.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "test", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "test de mail\r\n-- \r\nJames Hillyerd\r\nHillyerd IT Consulting\r\nwww.hillyerd.com\r\n \r\n\r\n", mime.Text)
  assert.Equal(t, "", mime.Html)
}

func Test05(t *testing.T) {
  msg := readMessage("05-multipart_text_html.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Jet Pilot - Mobile Game", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "Hi,\r\n\r\nHey guys, This is MY LATEST MOBILE GAME release.... As always... I need your support... so please Download the Game and give my game a  Positive Review and Rating. Every single review helps and is appreciated by me!\r\n\r\nDownload it on iPhones\r\nhttp://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/weSmAzda6GKAtGh9YZR0bA/0BZGk1K763AjSX4kaf826PJA\r\n\r\nDownload it on Google Play Store\r\nhttp://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/STyNnUbUqW7H9OAada0PuA/0BZGk1K763AjSX4kaf826PJA\r\n\r\nOr Play it Online on your PC\r\nhttp://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/E00ozmJ763ci65HI8QfLzejQ/0BZGk1K763AjSX4kaf826PJA\r\n\r\nThanks a lot for your support in helping my studio grow.\r\n\r\nBest Regards,\r\n\r\nNile Adams \r\nIn-charge of getting games everywhere...at FOG.COM\r\n\r\n\r\n\r\nYou got this email because\r\nyou are a member of our gaming website FOG.COM ( http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/VZVAUDRI1892I5dz0bwd3U2g/0BZGk1K763AjSX4kaf826PJA ). If you don't want to receive any more\r\nemails you can\r\nclick here to unsubscribe http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/LLhGMgRbpr4hKvADtLtmVw/0BZGk1K763AjSX4kaf826PJA\r\nFreeOnlineGames.com FZE, PO Box 191251, Dubai, UAE.\r\n\r\n", mime.Text)
  assert.Equal(t, "<html><head></head><body><table style=\"width: 100%; border: 0px; background: #9DDBE8;\" cellpadding=\"10\">\r\n<tbody>\r\n<tr>\r\n\t<td style=\"vertical-align: top;\">\r\n\t\t<center>\r\n\t\t<table style=\"text-align: center; width: 720px; border: 0px solid #C0C0C0; background: #fff; border-top: 5px solid #FF0033\" cellpadding=\"10\" cellspacing=\"0\">\r\n\t\t<tbody>\r\n\t\t<tr><td style=\"vertical-align: top;\">\r\n\t\t\t\t<table style=\"text-align: left; vertical-align: top; margin: 0; width: 704px; height: 82px; border: 0px solid #333;\" cellpadding=\"0\" cellspacing=\"1\">\r\n\t\t\t\t<tbody>\r\n\t\t\t\t<tr>\r\n\t\t\t\t\t<td style=\"align: center\">                    \r\n<span style=\"font-family: Georgia;\"><b>Hi, This is <span style=\"background-color: rgb(229, 224, 236);\"><span style=\"font-size: 20px;\">MY LATEST MOBILE GAME</span></span> release.... As always... I need your support... so please</b></span><span style=\"font-family: Georgia;\">\u00a0<span style=\"font-size: 18px;\"><b><span style=\"background-color: rgb(242, 195, 20);\">Download the Game</span>\u00a0</b></span>and give my game a\u00a0\u00a0<b><span style=\"background-color: rgb(242, 195, 20);\">Positive Review and Rating.</span></b>\u00a0Every single review helps and is appreciated by me!</span></td>\r\n\t\t\t\t</tr>\r\n\t\t\t\t</tbody>\r\n\t\t\t\t</table>\r\n\t\t\t\t<table style=\"width: 704px; border: 0px solid #333; background: #fff; align: center\" cellpadding=\"0\" cellspacing=\"0\">\r\n\t\t\t\t<tbody>\r\n\t\t\t\t<tr>\r\n\t\t\t\t\t<td style=\"width: 504px; vertical-align: top; text-align: left;\"><a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/fCXJKreuA5V5w7buIvj5qA/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402927829.jpg\" alt=\"3D Jet Pilot Flight Simulator Trailer\" border=\"0\"></a>\r\n\t\t\t\t\t\t\t\t \r\n\t\t\t\t\t</td>\r\n\t\t\t\t\t<td style=\"width: 200px; vertical-align: top;\">\r\n\t\t\t\t\t\t<table style=\"width: 200px; height: 240px; border: 0px;\" cellpadding=\"0\" cellspacing=\"0\">\r\n\t\t\t\t\t\t<tbody>\r\n\t\t\t\t\t\t<tr>\r\n\t\t\t\t\t\t\t<td style=\"width: 200px; height: 80px; vertical-align: top;\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402936886.jpg\"  border=\"0\"><br></td>\r\n\t\t\t\t\t\t</tr>\r\n\t\t\t\t\t\t<tr>\r\n\t\t\t\t\t\t\t<td style=\"width: 200px; height: 80px; vertical-align: top;\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402936890.jpg\"  border=\"0\"></td>\r\n\t\t\t\t\t\t</tr>\r\n\t\t\t\t\t\t<tr>\r\n\t\t\t\t\t\t\t<td style=\"width: 200px; height: 80px;vertical-align: top;\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402936895.jpg\"  border=\"0\"><br></td>\r\n\t\t\t\t\t\t</tr>\r\n\t\t\t\t\t\t</tbody>\r\n\t\t\t\t\t\t</table>\r\n</td></tr></tbody></table>\r\n\t\t\t\t<table style=\"text-align: right; width: 704px; height: 80px; border= 0; background: #fff;\" cellpadding=\"0\" cellspacing=\"6\">\r\n\t\t\t\t<tbody>\r\n\t\t\t\t<tr>\r\n\t\t\t\t\t<td>\r\n\t\t\t\t\t\t <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/E00ozmJ763ci65HI8QfLzejQ/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\" style=\"display: block;\"><img src=\"http://www.freeonlinegames.com/static/app-newsletter/Fog_Online_btn.png\" style=\"border: 0px none; width: 226px; height: 76px;\" alt=\"Play Online  On Your Computer\"></a>\r\n\t\t\t\t\t</td>\r\n\t\t\t\t\t<td>\r\n\t\t\t\t\t\t <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/weSmAzda6GKAtGh9YZR0bA/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\" style=\"display: block;\"><img src=\"http://www.freeonlinegames.com/static/app-newsletter/App_Store_btn.png\" style=\"border: 0px none; width: 226px; height: 76px;\" alt=\"Download and Play on Your iPhone\"></a>\r\n\t\t\t\t\t</td>\r\n\t\t\t\t\t<td>\r\n\t\t\t\t\t\t  <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/STyNnUbUqW7H9OAada0PuA/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\" style=\"display: block;\"><img src=\"http://www.freeonlinegames.com/static/app-newsletter/Google_Play_btn.png\" style=\"border: 0px none; width: 226px; height: 76px;\" alt=\"Download and Play from the Play Store\"></a>\r\n\t\t\t\t\t</td>\r\n\t\t\t\t</tr>\r\n\t\t\t\t</tbody>\r\n\t\t\t\t</table><p style=\"text-align: left;\">\r\n\t\t\t\t\t <strong><span style=\"font-family: Georgia;\">Thanks a lot for your support in helping my studio grow.</span></strong>\r\n\t\t\t\t</p><p style=\"text-align: left;\">\r\n\t\t\t\t\t regards,<br><strong>Nile Adams</strong><br><em>In-charge of getting games everywhere...at <a title=\"FOG.COM\" href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/lTfw8k7VxyLQa6IlbHjscw/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\">FOG.COM</a></em></p></td></tr></tbody></table><br><table style=\"width: 720px; text-align: center; border: 0;\" cellpadding=\"0\" cellspacing=\"0\">\r\n\t\t<tbody>\r\n\t\t<tr>\r\n\t\t\t<td>\r\n\t\t\t\t<p style=\"color:#FF6666; font-weight: normal; margin: 0; padding: 0; line-height: 20px; font-size: 14px;font-family: Courier, 'Monaco', \r\n\r\nmonospace;\">\r\n\t\t\t\t\t                               You got this email because you are a member of our gaming website <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/VZVAUDRI1892I5dz0bwd3U2g/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\">FOG.COM</a>.  If you don't want to receive any more emails you can click here to <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/LLhGMgRbpr4hKvADtLtmVw/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\">unsubscribe</a> \r\n\r\nFreeOnlineGames.com FZE, PO Box 191251, Dubai, UAE.\r\n\t\t\t\t</p>\r\n\t\t\t</td>\r\n\t\t</tr>\r\n\t\t</tbody>\r\n\t\t</table>\r\n</center>\r\n\t</td></tr></tbody></table></body></html><img src=\"http://newsletter.freeonlinegames.com/t/0BZGk1K763AjSX4kaf826PJA/sfRKKNqR6KA3bndCiFgF763A\" alt=\"\"/>\r\n\r\n\r\n", mime.Html)
}

func Test06(t *testing.T) {
  msg := readMessage("06-no_date.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Alerte 2e démarque : Soldes jusqu'à -70% et promotions", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "====================================\r\nAmazon.fr\r\n====================================\r\n\r\nChère cliente, cher client, \r\n\r\n3,2,1... C'est parti pour la 2ème démarque des Soldes* jusqu'a -70% . Et comme toujours, bénéficiez de la livraison gratuite dès 25 euros d'achats.\r\n\r\nCliquez ici\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_IntroBut/?node=1629578031\r\n\r\n====================================\r\n\r\nSoldes, 2e démarque Chaussures et Sacs\r\n Jusqu'à -70% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b1_t/?node=492180031\r\n\r\nSoldes, 2e démarque Vêtements\r\n Jusqu'à -70% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b2_t/?node=835623031\r\n\r\nSoldes, 2e démarque DVD & Blu-ray\r\n Jusqu'à -40% et petits prix\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b3_t/?node=1576933031\r\n\r\nSoldes, 2e démarque Sports\r\n Jusqu'à -50% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b4_t/?node=838098031\r\n\r\nSoldes, 2e démarque Informatique\r\nJusqu'à -70%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row2_b1_t/?node=2464308031\r\n\r\nSoldes, 2e démarque High-Tech\r\nDe -35% à -70%\r\nhttp://www.amazon.fr/gp/search/ref=pe_row2_b2_t/?ie=UTF8&page=1&rh=n%3A2472383031%2Cp_6%3AA1X6FK5RDHNB96%2Cn%3A%21425515031%2Cn%3A%21425514031%2Cn%3A13921051&bbn=2472383031\r\n\r\nSoldes, 2e démarque Bricolage\r\nJusqu'à - 40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row2_b3_t/?node=4066064031\r\n\r\nSoldes, 2e démarque Cuisine et Maison\r\nDe -10% à -40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row2_b4_t/?node=2848398031\r\n\r\nSoldes, 2e démarque Petit électroménager\r\nDe -10% à -30% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b1_t/?node=4811236031\r\n\r\nPromotions Jardin\r\nDe -10% à -30%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b2_t/?node=4933994031\r\n\r\nSoldes Montres\r\nJusqu'à -50%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b3_t/?node=1910645031\r\n\r\nSoldes Bijoux\r\nJusqu'à -50%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b4_t/?node=1910644031\r\n\r\nSoldes, 2e démarque Jeux Vidéo\r\nDe -10% à -60%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row4_b1_t/?node=4175523031\r\n\r\nSoldes, 2e démarque Bébé et Puériculture\r\nDe -10% à -60%\r\nhttp://www.amazon.fr/gp/search/ref=pe_row4_b2_t/?ie=UTF8&page=1&rh=n%3A4847720031%2Cp_6%3AA1X6FK5RDHNB96%2Cn%3A!425501031%2Cn%3A!425499031%2Cn%3A206617031&bbn=4847720031\r\n\r\nSoldes, 2e démarque Jeux et Jouets\r\nJusqu'à -40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row4_b3_t/?node=322086011&p_6=A1X6FK5RDHNB96\r\n\r\nSoldes, 2e démarque CD et Vinyles\r\nJusqu'à -40%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row4_b4_t/?node=4177238031\r\n\r\nSoldes Santé, bien-être et soins du corps\r\nDe -10% à -40%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b1_t/?node=4930110031&field-enc-merchantbin=A1X6FK5RDHNB96\r\n\r\nSoldes Beauté\r\nJusqu'à -40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b2_t/?node=4930081031&field-enc-merchantbin=A1X6FK5RDHNB96\r\n\r\n2 livres achetés = 1 offert\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b3_t/?node=301145\r\n\r\nSoldes Animalerie\r\nJusqu'à -20% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b4_t/?node=4916329031&field-enc-merchantbin=A1X6FK5RDHNB96\r\n\r\nSoldes, 2e démarque Bagages\r\nJusqu'à -50% et\u00a0promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row6_b1_t/?node=4908041031\r\n\r\nLiseuse Kindle Paperwhite\r\n Emportez votre bibliothèque en vacances\r\nhttp://www.amazon.fr/gp/product/ref=pe_row6_b2_t/?ASIN=B00JG8GBDM\r\n\r\nEbooks pour l'été\r\nNotre sélection d'ebooks à dévorer\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row6_b3_t/?node=4930884031\r\n\r\nLogiciels à petits prix\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row6_b4_t/?node=1630602031\r\n\r\nGlacières\r\npour auto\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b1_t/?node=2429738031\r\n\r\nPlus de 50 euros d'applis et jeux offerts\r\nsur l'App-Shop pour Android\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b2_t/?node=1661654031\r\n\r\nToutes les promotions\r\nEn téléchargement de musique\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b3_t/?node=212547031\r\n\r\nRetrouvez tous nos jeux vidéo disponibles\r\nen téléchargement\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b4_t/?node=2773594031\r\n\r\nProduits du quotidien\r\nÉconomisez 5% à 15% en programmant vos livraisons\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row8_b1_t/?node=424615031\r\n\r\nRecevez 10 euros\r\nen achetant un chèque-cadeau de 50 euros\r\nhttp://www.amazon.fr/gp/feature.html/ref=pe_row8_b2_t/?docId=1000807383\r\n\r\nPromotions et Offres Éclair\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row8_b3_t/?node=51375011\r\n\r\nAmazon Rachète vos Livres, Jeux vidéo et consoles\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row8_b4_t/?node=1325757031\r\n\r\n=====================================\r\n\r\n* Soldes et promotions du 25 juin 8h au 5 août 2014 inclus sur une grande sélection de produits expédiés et vendus par Amazon uniquement (cest-à-dire hors produits vendus par des vendeurs tiers sur la plateforme marketplace du site www.amazon.fr). Les démarques appliquées sont signalées sur les pages détaillées des produits concernés. Amazon se réserve le droit de retirer, suspendre ou modifier l'offre à tout moment. Les Conditions Générales de Vente du site www.amazon.fr s'appliquent à cette opération. \r\n \r\n\r\nCe message a été envoyé à james@hillyerd.com par Amazon EU S.à.r.l., RCS Luxembourg, B-101818, 5 Rue Plaetis, L-2338 Luxembourg, Grand- Duché du Luxembourg, (« Amazon.fr ») et Amazon Services Europe S.à.r.l., RCS Luxembourg, B-93815, 5 Rue Plaetis, L-2338 Luxembourg, Grand-Duché du Luxembourg. \r\n\r\nVeuillez noter que cet e-mail promotionnel a été envoyé à partir d'une adresse ne pouvant recevoir d'e-mails. Si vous souhaitez nous contacter, cliquez ici: http://www.amazon.fr/gp/browse.html/ref=pe_legal/?node=548536\r\n\r\nSi vous souhaitez ne plus recevoir ce type d'e-mail de la part d'Amazon.fr et Amazon Services Europe S.à.r.l., cliquez ici: http://www.amazon.fr//gp/gss/o/1h4LrjIuqSr2GGNEQYjqECpL.75WDdIp.cPuPkqCDsSefYZb1.qs3Odc149px-uGX \r\n\r\nA propos de nos conditions de vente : \r\n\r\nPour toute information concernant nos conditions de vente, consultez nos conditions générales de vente: http://www.amazon.fr/gp/help/customer/display.html?ie=UTF8&nodeId=548524 \r\n\r\nLes produits vendus par un vendeur Marketplace sont sujets aux conditions générales de ce dernier. \r\n\r\nLes informations et les prix mentionnés dans ce message peuvent faire l'objet de modifications entre l'envoi de cet e-mail et le moment où vous visitez notre site www.amazon.fr: http://www.amazon.fr/ref=pe_FootDomain. E-mail envoyé le 01/07/14 12:29 à 5h00 (GMT).\r\n", mime.Text)
  assert.Equal(t, "<00000146f178bcdb-2ce7f105-8f4e-4785-8ad5-ee9eeeec9a59-000000@eu-west-1.amazonses.com>", msg.Header.Get("MESSAGE-ID"))
  assert.Equal(t, msg.Header.Get("DATE"), "")
}

func Test07(t *testing.T) {
  msg := readMessage("07-no_message_id.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Pensez à proteger votre habitation ", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, msg.Header.Get("MESSAGE-ID"), "")
  assert.Equal(t, "Tue, 01 Jul 2014 13:55:20 +0200", msg.Header.Get("DATE"))
}

func Test08(t *testing.T) {
  msg := readMessage("08-no_to.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Maitrisez l'Anglais sur le bout des doigts !", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "<5b24ea5958601eb4eaad1d3d5e5cee61c4ea6d1947ad3dfe6ab52c0f157e60b3e2e27cbe8171c238@news.sprintmotorsport.com>", msg.Header.Get("MESSAGE-ID"))
  assert.Equal(t, "Tue, 01 Jul 2014 07:15:28 GMT", msg.Header.Get("DATE"))
}

func Test09(t *testing.T) {
  msg := readMessage("09-wrong_charset_in_part_header.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Confirmez l'inscription à la newsletter", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "<130528101156TB.13164@mssysweb07>", msg.Header.Get("MESSAGE-ID"))
  assert.Equal(t, "Tue, 28 May 2013 10:11:56 +0200", msg.Header.Get("DATE"))
  assert.Equal(t, "<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.0 Transitional//EN\" \"http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd\">\r\n<html xmlns=\"http://www.w3.org/1999/xhtml\" xml:lang=\"fr-CH\" lang=\"fr-CH\">\r\n<head>\r\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=iso-8859-1\" />\r\n<title>Confirmez votre inscription à la newsletter</title>\r\n</head>\r\n<body bgcolor=\"#82b04d\" style=\"font-family:Arial, Helvetica, sans-serif; text-align:left;\"><img src=\"http://news.groupon.ch/c/r?ACTION=hi&EMID=0A002VE0O520146CTBR&UID=6YZBVTUUF66IRQRQGLIQ\" border=\"0\" width=\"1\" height=\"1\"><table width=\"100%\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" bgcolor=\"#82b04d\" style=\"font-size:12px; color:#424242; text-align:left;\"><tr>\r\n<td align=\"center\"><table border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><tr>\r\n<td align=\"center\" valign=\"top\" style=\"padding-bottom:6px;\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:11px; color:#333333;\"><a target=\"_blank\" href=\"http://news.groupon.ch/c/dc?t=ol&p=bee.6YZBVTUUF66IRQRQGLIQ.TAT65WFFAC\" style=\"color:#333333; text-decoration:underline;\"><font style=\"color:#333333;\">version en ligne</font></a> &nbsp;|&nbsp; Ajouter \"info@news.groupon.ch\" à mon carnet d'adresses.</font></td>\r\n</tr>\r\n</table>\r\n</td>\r\n</tr><tr>\r\n<td align=\"center\"><table width=\"656\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><tr>\r\n<td align=\"left\"><table width=\"656\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" bgcolor=\"#333333\"><tr>\r\n<td width=\"10\" height=\"38\" align=\"left\" valign=\"top\" bgcolor=\"#333333\"><img src=\"http://static.ch.groupon-content.net/newsletter_ums/basic/frame_top_left_bg.gif\" width=\"10\" height=\"10\" alt=\"\" style=\"display:block;\" /></td>\r\n<td width=\"130\" height=\"38\" align=\"left\" valign=\"bottom\" bgcolor=\"#333333\"><a target=\"_blank\" href=\"http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QI004SOALE&UID=6YZBVTUUF66IRQRQGLIQ&newsletter_town_url=lucerne_fr&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/deals/lucerne_fr?nlp=&CID=CH_CRM_11_0_0_148&a=2338\" style=\"font-family:Arial, Helvetica, sans-serif; font-size:18px; color:#ffffff; text-decoration:none;\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:18px; color:#ffffff;\"><strong><img src=\"http://static.ch.groupon-content.net/newsletter_ums/basic/logo_groupon_top_fr_CH.gif\" width=\"130\" height=\"38\" border=\"0\" style=\"display:block;\" alt=\"Groupon\" /></strong></font></a></td>\r\n<td width=\"10\" height=\"38\" bgcolor=\"#333333\">&nbsp; </td>\r\n<td height=\"38\" align=\"left\" valign=\"middle\" bgcolor=\"#333333\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:16px; color:#ffffff;\"><a href=\"http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QI004SOALE&UID=6YZBVTUUF66IRQRQGLIQ&newsletter_town_url=lucerne_fr&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/deals/lucerne_fr?nlp=&CID=CH_CRM_11_0_0_148&a=2338\" style=\"color:#ffffff; text-decoration:none;\" target=\"_blank\"><font style=\"color:#ffffff;\"><strong>Confirmez votre inscription à la newsletter</strong></font></a></font></td>\r\n<td height=\"38\" align=\"right\" valign=\"middle\" bgcolor=\"#333333\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:11px; color:#ffffff;\"><a href=\"http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QI004SOALE&UID=6YZBVTUUF66IRQRQGLIQ&newsletter_town_url=lucerne_fr&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/deals/lucerne_fr?nlp=&CID=CH_CRM_11_0_0_148&a=2338\" style=\"color:#ffffff; text-decoration:none;\" target=\"_blank\"><font style=\"color:#ffffff;\">28.05.2013</font></a></font></td>\r\n<td width=\"24\" height=\"38\" align=\"right\" valign=\"top\" bgcolor=\"#333333\"><img src=\"http://static.ch.groupon-content.net/newsletter_ums/basic/frame_top_right_bg.gif\" width=\"10\" height=\"10\" alt=\"\" style=\"display:block;\" /></td>\r\n</tr>\r\n</table><table width=\"656\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" bgcolor=\"#ffffff\"><tr>\r\n<td width=\"10\" height=\"28\" align=\"left\" valign=\"top\"><img src=\"http://static.ch.groupon-content.net/newsletter_ums/basic/blue_line_10x22.gif\" width=\"10\" height=\"22\" border=\"0\" style=\"display:block;\" alt=\"\" /></td>\r\n<td width=\"130\" height=\"28\" align=\"left\" valign=\"top\"><a target=\"_blank\" href=\"http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QI004SOALE&UID=6YZBVTUUF66IRQRQGLIQ&newsletter_town_url=lucerne_fr&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/deals/lucerne_fr?nlp=&CID=CH_CRM_11_0_0_148&a=2338\" style=\"font-family:Arial, Helvetica, sans-serif; font-size:18px; color:#ffffff; text-decoration:none;\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:18px; color:#ffffff;\"><strong><img src=\"http://static.ch.groupon-content.net/newsletter_ums/basic/logo_groupon_bot_fr_CH.gif\" width=\"130\" height=\"22\" border=\"0\" style=\"display:block;\" alt=\"\" /></strong></font></a></td>\r\n<td width=\"516\" height=\"28\" align=\"left\" valign=\"top\"><img src=\"http://static.ch.groupon-content.net/newsletter_ums/basic/blue_line_10x22.gif\" width=\"516\" height=\"22\" border=\"0\" style=\"display:block;\" alt=\"\" /></td>\r\n</tr>\r\n</table><table width=\"656\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" bgcolor=\"#ffffff\"><tr>\r\n<td width=\"656\" align=\"center\" valign=\"top\"><table width=\"656\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><tr>\r\n<td width=\"656\" align=\"center\" valign=\"top\"><table width=\"630\" border=\"0\" cellpadding=\"9\" cellspacing=\"0\"><tr>\r\n<td width=\"612\" align=\"left\" valign=\"top\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:14px; color:#333333; line-height:1.3em;\"><strong>Confirmez votre inscription à la newsletter</strong><br /><br />Vous avez presque terminé ! Pour que nous puissions vous envoyer des offres personnalisées (bons plans de votre ville, shopping et voyages), veuillez confirmer votre inscription en cliquant sur le lien ci-dessous :</font></td>\r\n</tr>\r\n</table><table border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><tr>\r\n<td width=\"10\" height=\"44\" align=\"right\" valign=\"middle\" bgcolor=\"#ff7200\"><img src=\"http://static.ch.groupon-content.net/mail_img/global/basic/btn_orange_44_left.gif\" width=\"10\" height=\"44\" border=\"0\" alt=\"\" style=\"display:block;\" /></td>\r\n<td width=\"25\" height=\"44\" align=\"right\" valign=\"middle\" bgcolor=\"#ff7200\"><a href=\"http://www.groupon.ch/citynews/optin/2a1fc34c-3bc8-44f9-9526-7084f37d7fb0?nlp=&CID=CH_CRM_11_0_0_148&a=2338&qdoix=\" style=\"font-family:Arial, Helvetica, sans-serif; font-size:18px; color:#ffffff; text-decoration:none;\" target=\"_blank\"><font style=\"color:#ffffff;\"><img src=\"http://static.ch.groupon-content.net/mail_img/global/basic/btn_orange_arrow_right.gif\" width=\"25\" height=\"24\" border=\"0\" alt=\"&raquo;\" style=\"display:block;\" /></font></a></td>\r\n<td height=\"44\" align=\"center\" valign=\"middle\" bgcolor=\"#ff7200\"><a href=\"http://www.groupon.ch/citynews/optin/2a1fc34c-3bc8-44f9-9526-7084f37d7fb0?nlp=&CID=CH_CRM_11_0_0_148&a=2338&qdoix=\" style=\"font-family:Arial, Helvetica, sans-serif; font-size:18px; color:#ffffff; text-decoration:none;\" target=\"_blank\">&nbsp;&nbsp;<font style=\"color:#ffffff;\">Confirmer l'inscription</font>&nbsp;&nbsp;</a></td>\r\n<td width=\"10\" height=\"44\" align=\"left\" valign=\"middle\" bgcolor=\"#ff7200\"><img src=\"http://static.ch.groupon-content.net/mail_img/global/basic/btn_orange_44_right.gif\" width=\"10\" height=\"44\" border=\"0\" alt=\"\" style=\"display:block;\" /></td>\r\n</tr>\r\n</table><table width=\"630\" border=\"0\" cellpadding=\"9\" cellspacing=\"0\"><tr>\r\n<td width=\"612\" align=\"left\" valign=\"top\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:14px; color:#333333; line-height:1.3em;\">&nbsp;<br /><strong>Quelle est l'étape suivante ?</strong><br /><br />Une fois votre inscription confirmée, nous vous enverrons les meilleures offres de votre ville, shopping et voyages.<br /><br />Ces dernières seront personnalisées en fonction de votre profil : de votre emplacement géographique, de votre historique de navigation et d'achats et de toutes les informations que vous nous fournissez ou que nous récoltons grâce à vos visites sur notre site internet.<br />\r\n<br />Par la suite, vous pourrez demander à ne plus recevoir nos newsletters ou à personnaliser votre inscription en accédant directement à vos abonnements (vous pouvez y accéder facilement grâce au lien situé en bas de chacune des newsletters que nous envoyons).<br /><br />Si vous possédez déjà un compte Groupon, vous pouvez également gérer vos inscriptions en vous connectant directement à votre profil.<br /><br />Dans le cas où vous n'auriez pas encore confirmé votre inscription, nous nous permettrons de vous envoyer des emails de rappel pour que vous puissiez la compléter<br /><br />L'équipe Groupon</font></td>\r\n</tr>\r\n</table>\r\n</td>\r\n</tr>\r\n</table>\r\n</td>\r\n</tr>\r\n</table>\r\n</td>\r\n</tr><tr>\r\n<td height=\"6\" align=\"center\" bgcolor=\"#deeccc\">\r\n<!-- CNT_footer_service_fr_CH_104158_ums/subcontent_1 -->\r\n<table width=\"656\" border=\"0\" cellpadding=\"7\" cellspacing=\"0\">\r\n\t<tr>\r\n\t\t<td align=\"center\" valign=\"top\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:12px; color:#373737;\" face=\"Arial, Helvetica, sans-serif\"><strong>Pour toutes vos questions, contactez notre service membre</strong><br />Téléphone 044 278 77 16* &nbsp;|&nbsp; Email: servicemembre@groupon.ch &nbsp;|&nbsp; <a target=\"_blank\" href=\"http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QK00IEB627&UID=6YZBVTUUF66IRQRQGLIQ&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/faq?nlp=&CID=CH_CRM_11_0_0_148&a=2338\" style=\"color:#0891be; text-decoration:none;\"><font style=\"color:#0891be;\">FAQ</font></a> &nbsp;|&nbsp; <a target=\"_blank\" href=\"http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VE222B015CFPIA&UID=6YZBVTUUF66IRQRQGLIQ&facebook_id=pages%2Fgroup%2F312386792095&RED=http://www.facebook.com/pages/group/312386792095\" style=\"color:#0891be; text-decoration:none;\"><font style=\"color:#0891be;\">Facebook</font></a> &nbsp;|&nbsp; <a target=\"_blank\" href=\"htt\r\n p://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QL009FBMEL&UID=6YZBVTUUF66IRQRQGLIQ&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://blog.groupon.ch?nlp=&CID=CH_CRM_11_0_0_148&a=2338\" style=\"color:#0891be; text-decoration:none;\"><font style=\"color:#0891be;\">Blog</font></a><br /><font style=\"font-size:10px;\">Notre service membre est ouvert de 9h à 18h du lundi au vendredi et de 9h à 16h le samedi.</font></font></td>\r\n\t</tr>\r\n</table>\r\n<!-- /CNT_footer_service_fr_CH_104158_ums/subcontent_1 -->\r\n\r\n</td>\r\n</tr>\r\n</table>\r\n</td>\r\n</tr><tr>\r\n<td align=\"center\">\r\n<!-- CNT_footer_legal_fr_CH_104155_ums/subcontent_1 -->\r\n<table class=\"legal\" width=\"656\" border=\"0\" cellpadding=\"10\" cellspacing=\"0\">\r\n\t<tr>\r\n\t\t<td align=\"center\">\r\n\t\t\t<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\">\r\n\t\t\t\t<tr>\r\n\t\t\t\t\t<td align=\"center\" valign=\"top\"><font style=\"font-family:Arial, Helvetica, sans-serif; font-size:10px; color:#deeccc;\" face=\"Arial, Helvetica, sans-serif\">&copy; Groupon.ch est un service proposé par Groupon CH GmbH &nbsp;|&nbsp; <a target=\"_blank\" href=\"http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QJ00RGT2GJ&UID=6YZBVTUUF66IRQRQGLIQ&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/mentions_legales?nlp=&CID=CH_CRM_11_0_0_148&a=2338\" style=\"color:#deeccc;\"><font style=\"color:#deeccc;\">Mentions légales</font></a><br />PDG: Dr. Jens Hutzschenreuter, Mark S. Hoyt - Inscrit au registre du commerce du canton de Zurich, CH-020-4-042-334-0</font></td>\r\n\t\t\t\t</tr>\r\n\t\t\t</table>\r\n\t\t</td>\r\n\t</tr>\r\n</table>\r\n<!-- /CNT_footer_legal_fr_CH_104155_ums/subcontent_1 -->\r\n\r\n</td>\r\n</tr>\r\n</table>\r\n</body>\r\n</html>", mime.Html)
  assert.Equal(t, "Ajoutez \"info@news.groupon.ch\" à vos contacts pour être certain de recevoir les e-mails de CityDeal.\r\n\r\nSi cet e-mail ne s'affiche pas correctement, visualisez la version en ligne:\r\nhttp://news.groupon.ch/c/dc?t=ol&p=bee.6YZBVTUUF66IRQRQGLIQ.TAT65WFFAC\r\n\r\n--\r\n\r\nConfirmez votre inscription à la newsletter\r\n\r\n--\r\n\r\nVous avez presque terminé ! Pour que nous puissions vous envoyer des offres personnalisées (bons plans de votre ville, shopping et voyages), veuillez confirmer votre inscription en cliquant sur le lien ci-dessous :\r\n\r\nConfirmer l’inscription:\r\nhttp://www.groupon.ch/citynews/optin/2a1fc34c-3bc8-44f9-9526-7084f37d7fb0?nlp=&CID=CH_CRM_11_0_0_148&a=2338&qdoix=\r\n\r\n---\r\n\r\nQuelle est l’étape suivante ?\r\n\r\nUne fois votre inscription confirmée, nous vous enverrons les meilleures offres de votre ville, shopping et voyages.\r\n\r\nCes dernières seront personnalisées en fonction de votre profil : de votre emplacement géographique, de votre historique de navigation et d'achats et de toutes les informations que vous nous fournissez ou que nous récoltons grâce à vos visites sur notre site internet.\r\n\r\nPar la suite, vous pourrez demander à ne plus recevoir nos newsletters ou à personnaliser votre inscription en accédant directement à vos abonnements (vous pouvez y accéder facilement grâce au lien situé en bas de chacune des newsletters que nous envoyons).\r\n\r\nSi vous possédez déjà un compte Groupon, vous pouvez également gérer vos inscriptions en vous connectant directement à votre profil.\r\n\r\nDans le cas où vous n'auriez pas encore confirmé votre inscription, nous nous permettrons de vous envoyer des emails de rappel pour que vous puissiez la compléter\r\n\r\nL'équipe Groupon\r\n\r\n---\r\n\r\n\r\n---\r\n\r\nPour toutes vos questions, contactez notre service membre\r\n\r\nTéléphone : 044 278 77 16\r\nEmail : servicemembre@groupon.ch\r\n\r\nFAQ: http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QN00NEG74M&UID=6YZBVTUUF66IRQRQGLIQ&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/faq?nlp=&CID=CH_CRM_11_0_0_148&a=2338\r\nBlog: http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QO00E71SFM&UID=6YZBVTUUF66IRQRQGLIQ&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://blog.groupon.ch?nlp=&CID=CH_CRM_11_0_0_148&a=2338\r\n\r\n---\r\n\r\n\r\n\r\n--\r\n\r\n(c) Groupon.ch est un service proposé par Groupon CH GmbH\r\n\r\nMentions légales: http://news.groupon.ch/c/r?ACTION=ri&EMID=08G02VEL7QM010HODRU&UID=6YZBVTUUF66IRQRQGLIQ&url_param=CID%3DCH_CRM_11_0_0_148%26a%3D2338&RED=http://www.groupon.ch/mentions_legales?nlp=&CID=CH_CRM_11_0_0_148&a=2338\r\n\r\nPDG: Dr. Jens Hutzschenreuter, Mark S. Hoyt - Inscrit au registre du commerce du canton de Zurich, CH-020-4-042-334-0\r\n\r\n\r\n\r\n", mime.Text)

}

func Test10(t *testing.T) {
  msg := readMessage("10-wrong_transfer_encoding.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Digifit Fourth of July Special", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "<7bd20a2b-878e-442a-97b3-97040f7438cb@xtnvmta410.xt.local>", msg.Header.Get("MESSAGE-ID"))
  assert.Equal(t, "Thu, 03 Jul 2014 15:31:30 -0600", msg.Header.Get("DATE"))
}

func Test11(t *testing.T) {
  msg := readMessage("11-interleaved_text_parts.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Test", mime.GetHeader("Subject"))
  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 2, len(mime.Inlines))
  assert.Equal(t, "image-1.jpeg", mime.Inlines[0].FileName())
  assert.Equal(t, "image-2.jpeg", mime.Inlines[1].FileName())
  assert.Equal(t, "<26627E5E-DB8B-4568-8341-52C187D5E5F2@hillyerd.com>", msg.Header.Get("MESSAGE-ID"))
  assert.Equal(t, "Fri, 4 Jul 2014 11:19:07 +0200", msg.Header.Get("DATE"))
  assert.Equal(t, "", mime.Html)
  assert.Equal(t, "Grunt\r\n\r\n\n--\n\r\n\r\nJames Hillyerd\r\n\r\nYoupi\r\n\r\n\n--\n\r\n\r\n\r\nEndGrunt", mime.Text)
}

func Test12(t *testing.T) {
  msg := readMessage("12-latin_1_from.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Votre mutuelle à partir de 6.62 euros, devis gratuit et sans engagement", mime.GetHeader("Subject"))
  fmt.Printf("Subject: %v  \n", mime.GetHeader("Subject"))
  fmt.Printf("From: %v  \n", mime.GetHeader("From"))
  fmt.Printf("To: %v  \n", mime.GetHeader("To"))

  assert.Equal(t, 0, len(mime.Attachments))
  assert.Equal(t, 0, len(mime.Inlines))
  assert.Equal(t, "<e558ae7608c0d4e7a4e55f003c47be9b6bb9cf5ad3e26c0318d98dfaac977090e49fb5f1a19899a8@mon.eshopdeal.com>", msg.Header.Get("MESSAGE-ID"))
  assert.Equal(t, "Wed, 23 Jul 2014 04:37:56 GMT", msg.Header.Get("DATE"))
}

func Test13(t *testing.T) {
  msg := readMessage("13-B64-subject.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Smartphones à prix cassés", mime.GetHeader("Subject"))
}

func Test14(t *testing.T) {
  msg := readMessage("14-windows-1252_full.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "lettre de nouvelles : sortie moto - simulateur", mime.GetHeader("Subject"))
  assert.Equal(t, "COD106 - Centre de formation routière <info@cod106.ch>", mime.GetHeader("From"))
  assert.Equal(t, "Chers élèves et amis de COD106,\r\n\r\nVoilà, les beaux jours sont arrivés et notre sortie moto approche à grand pas!\r\n\r\nComme l'année passée, nous aurons un beau parcours avec plusieurs cols dans les préalpes.\r\n\r\nDate: samedi 23 août 2014\r\n\r\nDépart à 8h00 et retour en fin de journée.\r\nInscription gratuite, invitez vos amis !\r\nRepas de midi au restaurant, à charge des participants.\r\n\r\n\r\n\r\nMerci de vous inscrire dès que possible pour que nous puissions prévoir le nombre de personnes pour le repas !\r\nSur www.cod106.ch,  sur Facebook, ou par téléphone ou SMS au 079 5000 106.\r\n\r\nNous nous réjouissons de vous revoir à cette occasion !\r\n\r\nNouveau: Conduite sur simulateur!\r\n\r\nDès maintenant, chez COD106, vous pouvez suivre des cours sur nos simulateurs spécialement conçu pour l'apprentissage de la conduite.\r\n\r\nPas besoin de permis!\r\nPendant que vous préparez votre théorie, vous pouvez déjà commencer la pratique!\r\nEn plus, cela revient à 50 % du prix par rapport aux leçons en voiture.\r\nAlors, venez essayer !\r\n\r\nForfaits possibles: premiers secours, théorie et simulateur.\r\nPour plus de renseignements, vous pouvez aller voir sur notre site internet www.cod106.ch.\r\n\r\nN'oubliez pas notre nouvelle adresse: av. Léopold-Robert 132, tout près de la Coop des Entilles. \r\n\r\nSalutations estivales!\r\n\r\nvotre team COD106\r\n\r\nSi vous ne souhaitez plus recevoir de nouvelles concernant nos prochains cours de perfectionnement et sorties, veuillez simplement nous le signaler par mail à notre adresse habituelle: info@cod106.ch.\r\n", mime.Text)
  assert.Equal(t, "<HEAD>\r\n<META content=\"text/html; charset=windows-1252\" http-equiv=Content-Type>\r\n<META name=GENERATOR content=\"MSHTML 8.00.6001.23588\">\r\n<STYLE></STYLE>\r\n</HEAD>\r\n<BODY bgColor=#ffffff text=#000000>\r\n<P>\r\n<P></P>\r\n<P><FONT size=4 face=\"Arial, Helvetica, sans-serif\">Chers élèves et amis de COD106,</FONT></P>\r\n<P><FONT face=\"Arial, Helvetica, sans-serif\">Voilà, les beaux jours sont arrivés et notre sortie moto approche à grand pas!</FONT></P>\r\n<P><FONT face=\"Arial, Helvetica, sans-serif\">Comme l'année passée, nous aurons un beau parcours avec plusieurs cols dans les préalpes.</FONT></P>\r\n<P><FONT face=Arial><STRONG>Date: samedi&nbsp;23 août 2014</STRONG></FONT></P>\r\n<P><FONT face=\"Arial, Helvetica, sans-serif\">Départ à 8h00 et retour en fin de journée.<BR></FONT><FONT face=\"Arial, Helvetica, sans-serif\">Inscription gratuite, invitez vos amis !<BR>Repas de midi au restaurant, à charge des participants.</FONT></P><FONT face=\"Arial, Helvetica, sans-serif\">\r\n<P><IMG border=0 hspace=0 alt=\"\" align=baseline src=\"cid:7371048031-1\"><BR><BR>Merci de vous inscrire</FONT><FONT face=\"Arial, Helvetica, sans-serif\"> dès que possible pour que nous puissions prévoir le nombre de personnes pour le repas !<BR></FONT><FONT face=Arial>Sur </FONT><A href=\"http://www.cod106.ch/\"><FONT face=Arial>www.cod106.ch</FONT></A>,&nbsp;<FONT face=Arial> sur </FONT><A href=\"https://www.facebook.com/events/431274390339680/?ref=5\"><FONT face=Arial>Facebook</FONT></A><FONT face=Arial>, ou par téléphone ou SMS au</FONT><FONT face=\"Arial, Helvetica, sans-serif\">&nbsp;079 5000 106.</FONT></P>\r\n<P><FONT face=Arial><FONT face=\"Arial, Helvetica, sans-serif\">Nous nous réjouissons de vous revoir à cette occasion !</FONT></FONT></P>\r\n<P>\r\n<HR id=false>\r\n\r\n<P></P>\r\n<P><FONT size=4 face=Arial>Nouveau: Conduite sur simulateur!</FONT></P><FONT size=4 face=Arial>\r\n<P><FONT size=3>Dès maintenant, chez COD106, vous pouvez suivre des cours sur nos simulateurs spécialement conçu pour l'apprentissage de la conduite.</FONT></P>\r\n<P><FONT size=3>Pas besoin de permis!<BR>Pendant que vous préparez votre théorie, vous pouvez déjà commencer la pratique!<BR>En plus, cela revient à 50 % du prix par rapport aux leçons en voiture.<BR>Alors, venez essayer !</FONT></P>\r\n<P><FONT size=3>Forfaits possibles: premiers secours, théorie et simulateur.<BR>Pour plus de renseignements, vous pouvez aller voir sur notre site internet <A href=\"http://www.cod106.ch/\"><FONT face=Arial>www.cod106.ch</FONT></A>.</FONT></P>\r\n<P><FONT size=3></FONT></P><FONT size=3>\r\n<HR id=false>\r\nN'oubliez pas notre nouvelle adresse: av. Léopold-Robert 132, tout près de la Coop des Entilles. </FONT>\r\n<P></P>\r\n<P><FONT size=3>Salutations estivales!</FONT></P>\r\n<P><FONT face=Arial><FONT size=3>votre team COD106<BR></FONT></P></FONT><FONT face=Arial><STRONG>\r\n<HR SIZE=2 width=\"100%\">\r\n</STRONG></FONT>\r\n<DIV><SMALL><FONT size=2 face=Arial><STRONG>Si vous ne souhaitez plus recevoir de nouvelles concernant nos prochains cours de perfectionnement et sorties, veuillez simplement nous le signaler par mail&nbsp;à notre adresse habituelle: </STRONG></FONT><A class=moz-txt-link-abbreviated href=\"mailto:info@cod106.ch\" target=blank moz-do-not-send=\"true\"><FONT size=2 face=Arial><STRONG>info@cod106.ch</STRONG></FONT></A><FONT size=2 face=Arial><STRONG>.</STRONG></FONT></SMALL><BR></DIV>\r\n<BLOCKQUOTE>\r\n<BLOCKQUOTE><FONT face=Arial></FONT></BLOCKQUOTE></BLOCKQUOTE></FONT></BODY>", mime.Html)
}

func Test15(t *testing.T) {
  msg := readMessage("15-windows-1251_full.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "Новая система в индустрии рекламы", mime.GetHeader("Subject"))
  assert.Equal(t, "Новая система в индустрии рекламы", mime.Text)
  assert.Equal(t, "Новая система в индустрии рекламы", mime.Html)
}

func Test16(t *testing.T) {
  msg := readMessage("16-latin_1_text_body.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, "\r\nj'ai vu que tu as supprimé tous les magasins dans les marchés, merci \r\nbeaucoup!\r\n", mime.Text, "Should decode latin1 body")
}


// readMessage is a test utility function to fetch a mail.Message object.
func readMessage(filename string) *mail.Message {
  // Open test email for parsing
  raw, err := os.Open(filepath.Join("test-data", "mail", filename))
  if err != nil {
    panic(fmt.Sprintf("Failed to open test data: %v", err))
  }

  // Parse email into a mail.Message object like we do
  reader := bufio.NewReader(raw)
  msg, err := mail.ReadMessage(reader)
  if err != nil {
    panic(fmt.Sprintf("Failed to read message: %v", err))
  }

  return msg
}
