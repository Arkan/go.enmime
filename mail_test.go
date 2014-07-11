package enmime

import (
  "bufio"
  "bytes"
  "fmt"
  "net/mail"
  "os"
  "path/filepath"
  "testing"

  "github.com/stretchr/testify/assert"
)

func TestIdentifySinglePart(t *testing.T) {
  msg := readMessage("non-mime.raw")
  assert.False(t, IsMultipartMessage(msg), "Failed to identify non-multipart message")
}

func TestIdentifyMultiPart(t *testing.T) {
  msg := readMessage("html-mime-inline.raw")
  assert.True(t, IsMultipartMessage(msg), "Failed to identify multipart MIME message")
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

  assert.Contains(t, mime.Text, "This is a test mailing")
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

  assert.Equal(t, mime.Text, "Test of text section")
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
  assert.Equal(t, mime.Html, "", "Html attachment is not for display")
  assert.Equal(t, len(mime.Inlines), 0, "Should have no inlines")
  assert.Equal(t, len(mime.Attachments), 1, "Should have a single attachment")
  assert.Equal(t, mime.Attachments[0].FileName(), "test.html", "Attachment should have correct filename")
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
  assert.Equal(t, len(mime.Inlines), 1, "Should have one inline")
  assert.Equal(t, len(mime.Attachments), 0, "Should have no attachments")
  assert.Equal(t, mime.Inlines[0].FileName(), "favicon.png", "Inline should have correct filename")
  assert.True(t, bytes.HasPrefix(mime.Inlines[0].Content(), []byte{0x89, 'P', 'N', 'G'}),
    "Content should be PNG image")
}

func TestParseNestedHeaders(t *testing.T) {
  msg := readMessage("html-mime-inline.raw")
  mime, err := ParseMIMEBody(msg)

  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }

  assert.Equal(t, len(mime.Inlines), 1, "Should have one inline")
  assert.Equal(t, mime.Inlines[0].FileName(), "favicon.png", "Inline should have correct filename")
  assert.Equal(t, mime.Inlines[0].Header().Get("Content-Id"), "<8B8481A2-25CA-4886-9B5A-8EB9115DD064@skynet>", "Inline should have a Content-Id header")
}

func TestParseEncodedSubject(t *testing.T) {
  // Even non-MIME messages should support encoded-words in headers
  msg := readMessage("qp-ascii-header.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  assert.Equal(t, mime.GetHeader("Subject"), "Test QP Subject!")

  // Test UTF-8 subject line
  msg = readMessage("qp-utf8-header.raw")
  mime, err = ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse MIME: %v", err)
  }
  assert.Equal(t, mime.GetHeader("Subject"), "MIME UTF8 Test \u00a2 More Text")
}

func TestSparrow(t *testing.T) {
  msg := readMessage("sparrow.raw")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  assert.Equal(t, mime.GetHeader("Subject"), "LOLZ3")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, mime.Text, "Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum\n\n--\nJames\n\n")
  assert.Equal(t, mime.Html, "                <div>\n                    <span style=\"font-family: Arial, Helvetica, sans; font-size: 11px; line-height: 14px; text-align: justify;\">Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum</span>\n                </div>\n                <div><div><br></div><div>--&nbsp;</div><div>Florian Bertholin</div><div><br></div></div>\n")
}

// func Test01(t *testing.T) {
//   msg := readMessage("01-body_is_an_attachment.eml")
//   mime, err := ParseMIMEBody(msg)
//   if err != nil {
//     t.Fatalf("Failed to parse non-MIME: %v", err)
//   }
//   _ = mime
//   assert.Equal(t, mime.GetHeader("Subject"), "Test photo")
//   assert.Equal(t, len(mime.Attachments), 0)
//   assert.Equal(t, len(mime.Inlines), 0)
//   assert.Equal(t, mime.Text, "")
//   assert.Equal(t, mime.Html, "")
// }

func Test02(t *testing.T) {
  msg := readMessage("02-inline_pictures.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "test text parts")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 2)
  assert.Equal(t, mime.Text, "youpi\r\n\r\n\r\n\r\n\r\ntrop bi1\r\n\r\n\r\n\r\n\r\nburp\r\n\r\n")
  assert.Equal(t, mime.Html, "<html><head><meta http-equiv=\"Content-Type\" content=\"text/html charset=us-ascii\"></head><body style=\"word-wrap: break-word; -webkit-nbsp-mode: space; -webkit-line-break: after-white-space;\">youpi<div><br></div><div><img height=\"250\" width=\"210\" apple-width=\"yes\" apple-height=\"yes\" apple-inline=\"yes\" id=\"F4FA4AF9-956C-43CE-9E03-2CA6D2A59952\" src=\"cid:F2E213E4-D044-4BEF-982D-12FCB8701F31@hillyerd.com\"></div><div><br></div><div><br></div><div>trop bi1</div><div><br></div><div><img height=\"235\" width=\"250\" apple-width=\"yes\" apple-height=\"yes\" apple-inline=\"yes\" id=\"997EE7FA-0B89-4977-9D86-9117B1295E10\" src=\"cid:E438E104-EC3A-4AF5-B985-2148A44E48AE@hillyerd.com\"></div><div><br></div><div><br></div><div>burp</div><div><br></div></body></html>")
}

func Test03(t *testing.T) {
  msg := readMessage("03-monopart_html_only.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Test mail monopart HTML only")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, mime.Text, "")
  assert.Equal(t, mime.Html, "<html><head>\r\n<meta content=\"text/html; charset=ISO-8859-1\" http-equiv=\"Content-Type\">\r\n</head><body bgcolor=\"#FFFFFF\" text=\"#000000\">\r\nCeci est une part HTML. Il n'y a pas de part TEXT<br>\r\n  <br>\r\n  <span style=\"font-weight: bold;\">YOUPIIII<br>\r\n    <br>\r\n  </span>Voila.<span style=\"font-weight: bold;\"><span\r\nstyle=\"font-weight: bold;\"><br>\r\n      <br>\r\n    </span></span>\r\n</body>\r\n</html>\r\n\r\n")
}

func Test04(t *testing.T) {
  msg := readMessage("04-monopart_plain_text.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "test")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, mime.Text, "test de mail\r\n-- \r\nJames Hillyerd\r\nHillyerd IT Consulting\r\nwww.hillyerd.com\r\n \r\n\r\n")
  assert.Equal(t, mime.Html, "")
}

func Test05(t *testing.T) {
  msg := readMessage("05-multipart_text_html.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Jet Pilot - Mobile Game")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, mime.Text, "Hi,\r\n\r\nHey guys, This is MY LATEST MOBILE GAME release.... As always... I need your support... so please Download the Game and give my game a  Positive Review and Rating. Every single review helps and is appreciated by me!\r\n\r\nDownload it on iPhones\r\nhttp://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/weSmAzda6GKAtGh9YZR0bA/0BZGk1K763AjSX4kaf826PJA\r\n\r\nDownload it on Google Play Store\r\nhttp://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/STyNnUbUqW7H9OAada0PuA/0BZGk1K763AjSX4kaf826PJA\r\n\r\nOr Play it Online on your PC\r\nhttp://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/E00ozmJ763ci65HI8QfLzejQ/0BZGk1K763AjSX4kaf826PJA\r\n\r\nThanks a lot for your support in helping my studio grow.\r\n\r\nBest Regards,\r\n\r\nNile Adams \r\nIn-charge of getting games everywhere...at FOG.COM\r\n\r\n\r\n\r\nYou got this email because\r\nyou are a member of our gaming website FOG.COM ( http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/VZVAUDRI1892I5dz0bwd3U2g/0BZGk1K763AjSX4kaf826PJA ). If you don't want to receive any more\r\nemails you can\r\nclick here to unsubscribe http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/LLhGMgRbpr4hKvADtLtmVw/0BZGk1K763AjSX4kaf826PJA\r\nFreeOnlineGames.com FZE, PO Box 191251, Dubai, UAE.\r\n\r\n")
  assert.Equal(t, mime.Html, "<html><head></head><body><table style=\"width: 100%; border: 0px; background: #9DDBE8;\" cellpadding=\"10\">\r\n<tbody>\r\n<tr>\r\n\t<td style=\"vertical-align: top;\">\r\n\t\t<center>\r\n\t\t<table style=\"text-align: center; width: 720px; border: 0px solid #C0C0C0; background: #fff; border-top: 5px solid #FF0033\" cellpadding=\"10\" cellspacing=\"0\">\r\n\t\t<tbody>\r\n\t\t<tr><td style=\"vertical-align: top;\">\r\n\t\t\t\t<table style=\"text-align: left; vertical-align: top; margin: 0; width: 704px; height: 82px; border: 0px solid #333;\" cellpadding=\"0\" cellspacing=\"1\">\r\n\t\t\t\t<tbody>\r\n\t\t\t\t<tr>\r\n\t\t\t\t\t<td style=\"align: center\">                    \r\n<span style=\"font-family: Georgia;\"><b>Hi, This is <span style=\"background-color: rgb(229, 224, 236);\"><span style=\"font-size: 20px;\">MY LATEST MOBILE GAME</span></span> release.... As always... I need your support... so please</b></span><span style=\"font-family: Georgia;\">\u00a0<span style=\"font-size: 18px;\"><b><span style=\"background-color: rgb(242, 195, 20);\">Download the Game</span>\u00a0</b></span>and give my game a\u00a0\u00a0<b><span style=\"background-color: rgb(242, 195, 20);\">Positive Review and Rating.</span></b>\u00a0Every single review helps and is appreciated by me!</span></td>\r\n\t\t\t\t</tr>\r\n\t\t\t\t</tbody>\r\n\t\t\t\t</table>\r\n\t\t\t\t<table style=\"width: 704px; border: 0px solid #333; background: #fff; align: center\" cellpadding=\"0\" cellspacing=\"0\">\r\n\t\t\t\t<tbody>\r\n\t\t\t\t<tr>\r\n\t\t\t\t\t<td style=\"width: 504px; vertical-align: top; text-align: left;\"><a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/fCXJKreuA5V5w7buIvj5qA/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402927829.jpg\" alt=\"3D Jet Pilot Flight Simulator Trailer\" border=\"0\"></a>\r\n\t\t\t\t\t\t\t\t \r\n\t\t\t\t\t</td>\r\n\t\t\t\t\t<td style=\"width: 200px; vertical-align: top;\">\r\n\t\t\t\t\t\t<table style=\"width: 200px; height: 240px; border: 0px;\" cellpadding=\"0\" cellspacing=\"0\">\r\n\t\t\t\t\t\t<tbody>\r\n\t\t\t\t\t\t<tr>\r\n\t\t\t\t\t\t\t<td style=\"width: 200px; height: 80px; vertical-align: top;\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402936886.jpg\"  border=\"0\"><br></td>\r\n\t\t\t\t\t\t</tr>\r\n\t\t\t\t\t\t<tr>\r\n\t\t\t\t\t\t\t<td style=\"width: 200px; height: 80px; vertical-align: top;\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402936890.jpg\"  border=\"0\"></td>\r\n\t\t\t\t\t\t</tr>\r\n\t\t\t\t\t\t<tr>\r\n\t\t\t\t\t\t\t<td style=\"width: 200px; height: 80px;vertical-align: top;\"><img src=\"http://newsletter.freeonlinegames.com/uploads/1402936895.jpg\"  border=\"0\"><br></td>\r\n\t\t\t\t\t\t</tr>\r\n\t\t\t\t\t\t</tbody>\r\n\t\t\t\t\t\t</table>\r\n</td></tr></tbody></table>\r\n\t\t\t\t<table style=\"text-align: right; width: 704px; height: 80px; border= 0; background: #fff;\" cellpadding=\"0\" cellspacing=\"6\">\r\n\t\t\t\t<tbody>\r\n\t\t\t\t<tr>\r\n\t\t\t\t\t<td>\r\n\t\t\t\t\t\t <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/E00ozmJ763ci65HI8QfLzejQ/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\" style=\"display: block;\"><img src=\"http://www.freeonlinegames.com/static/app-newsletter/Fog_Online_btn.png\" style=\"border: 0px none; width: 226px; height: 76px;\" alt=\"Play Online  On Your Computer\"></a>\r\n\t\t\t\t\t</td>\r\n\t\t\t\t\t<td>\r\n\t\t\t\t\t\t <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/weSmAzda6GKAtGh9YZR0bA/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\" style=\"display: block;\"><img src=\"http://www.freeonlinegames.com/static/app-newsletter/App_Store_btn.png\" style=\"border: 0px none; width: 226px; height: 76px;\" alt=\"Download and Play on Your iPhone\"></a>\r\n\t\t\t\t\t</td>\r\n\t\t\t\t\t<td>\r\n\t\t\t\t\t\t  <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/STyNnUbUqW7H9OAada0PuA/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\" style=\"display: block;\"><img src=\"http://www.freeonlinegames.com/static/app-newsletter/Google_Play_btn.png\" style=\"border: 0px none; width: 226px; height: 76px;\" alt=\"Download and Play from the Play Store\"></a>\r\n\t\t\t\t\t</td>\r\n\t\t\t\t</tr>\r\n\t\t\t\t</tbody>\r\n\t\t\t\t</table><p style=\"text-align: left;\">\r\n\t\t\t\t\t <strong><span style=\"font-family: Georgia;\">Thanks a lot for your support in helping my studio grow.</span></strong>\r\n\t\t\t\t</p><p style=\"text-align: left;\">\r\n\t\t\t\t\t regards,<br><strong>Nile Adams</strong><br><em>In-charge of getting games everywhere...at <a title=\"FOG.COM\" href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/lTfw8k7VxyLQa6IlbHjscw/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\">FOG.COM</a></em></p></td></tr></tbody></table><br><table style=\"width: 720px; text-align: center; border: 0;\" cellpadding=\"0\" cellspacing=\"0\">\r\n\t\t<tbody>\r\n\t\t<tr>\r\n\t\t\t<td>\r\n\t\t\t\t<p style=\"color:#FF6666; font-weight: normal; margin: 0; padding: 0; line-height: 20px; font-size: 14px;font-family: Courier, 'Monaco', \r\n\r\nmonospace;\">\r\n\t\t\t\t\t                               You got this email because you are a member of our gaming website <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/VZVAUDRI1892I5dz0bwd3U2g/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\">FOG.COM</a>.  If you don't want to receive any more emails you can click here to <a href=\"http://newsletter.freeonlinegames.com/l/sfRKKNqR6KA3bndCiFgF763A/LLhGMgRbpr4hKvADtLtmVw/0BZGk1K763AjSX4kaf826PJA\" target=\"_blank\">unsubscribe</a> \r\n\r\nFreeOnlineGames.com FZE, PO Box 191251, Dubai, UAE.\r\n\t\t\t\t</p>\r\n\t\t\t</td>\r\n\t\t</tr>\r\n\t\t</tbody>\r\n\t\t</table>\r\n</center>\r\n\t</td></tr></tbody></table></body></html><img src=\"http://newsletter.freeonlinegames.com/t/0BZGk1K763AjSX4kaf826PJA/sfRKKNqR6KA3bndCiFgF763A\" alt=\"\"/>\r\n\r\n\r\n")
}

func Test06(t *testing.T) {
  msg := readMessage("06-no_date.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Alerte 2e démarque : Soldes jusqu'à -70% et promotions")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, mime.Text, "====================================\r\nAmazon.fr\r\n====================================\r\n\r\nCh\xe8re cliente, cher client, \r\n\r\n3,2,1... C'est parti pour la 2\xe8me d\xe9marque des Soldes* jusqu'a -70% . Et comme toujours, b\xe9n\xe9ficiez de la livraison gratuite d\xe8s 25 euros d'achats.\r\n\r\nCliquez ici\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_IntroBut/?node=1629578031\r\n\r\n====================================\r\n\r\nSoldes, 2e d\xe9marque Chaussures et Sacs\r\n Jusqu'\xe0 -70% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b1_t/?node=492180031\r\n\r\nSoldes, 2e d\xe9marque V\xeatements\r\n Jusqu'\xe0 -70% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b2_t/?node=835623031\r\n\r\nSoldes, 2e d\xe9marque DVD & Blu-ray\r\n Jusqu'\xe0 -40% et petits prix\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b3_t/?node=1576933031\r\n\r\nSoldes, 2e d\xe9marque Sports\r\n Jusqu'\xe0 -50% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row1_b4_t/?node=838098031\r\n\r\nSoldes, 2e d\xe9marque Informatique\r\nJusqu'\xe0 -70%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row2_b1_t/?node=2464308031\r\n\r\nSoldes, 2e d\xe9marque High-Tech\r\nDe -35% \xe0 -70%\r\nhttp://www.amazon.fr/gp/search/ref=pe_row2_b2_t/?ie=UTF8&page=1&rh=n%3A2472383031%2Cp_6%3AA1X6FK5RDHNB96%2Cn%3A%21425515031%2Cn%3A%21425514031%2Cn%3A13921051&bbn=2472383031\r\n\r\nSoldes, 2e d\xe9marque Bricolage\r\nJusqu'\xe0 - 40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row2_b3_t/?node=4066064031\r\n\r\nSoldes, 2e d\xe9marque Cuisine et Maison\r\nDe -10% \xe0 -40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row2_b4_t/?node=2848398031\r\n\r\nSoldes, 2e d\xe9marque Petit \xe9lectrom\xe9nager\r\nDe -10% \xe0 -30% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b1_t/?node=4811236031\r\n\r\nPromotions Jardin\r\nDe -10% \xe0 -30%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b2_t/?node=4933994031\r\n\r\nSoldes Montres\r\nJusqu'\xe0 -50%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b3_t/?node=1910645031\r\n\r\nSoldes Bijoux\r\nJusqu'\xe0 -50%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row3_b4_t/?node=1910644031\r\n\r\nSoldes, 2e d\xe9marque Jeux Vid\xe9o\r\nDe -10% \xe0 -60%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row4_b1_t/?node=4175523031\r\n\r\nSoldes, 2e d\xe9marque B\xe9b\xe9 et Pu\xe9riculture\r\nDe -10% \xe0 -60%\r\nhttp://www.amazon.fr/gp/search/ref=pe_row4_b2_t/?ie=UTF8&page=1&rh=n%3A4847720031%2Cp_6%3AA1X6FK5RDHNB96%2Cn%3A!425501031%2Cn%3A!425499031%2Cn%3A206617031&bbn=4847720031\r\n\r\nSoldes, 2e d\xe9marque Jeux et Jouets\r\nJusqu'\xe0 -40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row4_b3_t/?node=322086011&p_6=A1X6FK5RDHNB96\r\n\r\nSoldes, 2e d\xe9marque CD et Vinyles\r\nJusqu'\xe0 -40%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row4_b4_t/?node=4177238031\r\n\r\nSoldes Sant\xe9, bien-\xeatre et soins du corps\r\nDe -10% \xe0 -40%\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b1_t/?node=4930110031&field-enc-merchantbin=A1X6FK5RDHNB96\r\n\r\nSoldes Beaut\xe9\r\nJusqu'\xe0 -40% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b2_t/?node=4930081031&field-enc-merchantbin=A1X6FK5RDHNB96\r\n\r\n2 livres achet\xe9s = 1 offert\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b3_t/?node=301145\r\n\r\nSoldes Animalerie\r\nJusqu'\xe0 -20% et promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row5_b4_t/?node=4916329031&field-enc-merchantbin=A1X6FK5RDHNB96\r\n\r\nSoldes, 2e d\xe9marque Bagages\r\nJusqu'\xe0 -50% et\xa0promotions\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row6_b1_t/?node=4908041031\r\n\r\nLiseuse Kindle Paperwhite\r\n Emportez votre biblioth\xe8que en vacances\r\nhttp://www.amazon.fr/gp/product/ref=pe_row6_b2_t/?ASIN=B00JG8GBDM\r\n\r\nEbooks pour l'\xe9t\xe9\r\nNotre s\xe9lection d'ebooks \xe0 d\xe9vorer\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row6_b3_t/?node=4930884031\r\n\r\nLogiciels \xe0 petits prix\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row6_b4_t/?node=1630602031\r\n\r\nGlaci\xe8res\r\npour auto\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b1_t/?node=2429738031\r\n\r\nPlus de 50 euros d'applis et jeux offerts\r\nsur l'App-Shop pour Android\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b2_t/?node=1661654031\r\n\r\nToutes les promotions\r\nEn t\xe9l\xe9chargement de musique\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b3_t/?node=212547031\r\n\r\nRetrouvez tous nos jeux vid\xe9o disponibles\r\nen t\xe9l\xe9chargement\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row7_b4_t/?node=2773594031\r\n\r\nProduits du quotidien\r\n\xc9conomisez 5% \xe0 15% en programmant vos livraisons\r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row8_b1_t/?node=424615031\r\n\r\nRecevez 10 euros\r\nen achetant un ch\xe8que-cadeau de 50 euros\r\nhttp://www.amazon.fr/gp/feature.html/ref=pe_row8_b2_t/?docId=1000807383\r\n\r\nPromotions et Offres \xc9clair\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row8_b3_t/?node=51375011\r\n\r\nAmazon Rach\xe8te vos Livres, Jeux vid\xe9o et consoles\r\n \r\nhttp://www.amazon.fr/gp/browse.html/ref=pe_row8_b4_t/?node=1325757031\r\n\r\n=====================================\r\n\r\n* Soldes et promotions du 25 juin 8h au 5 ao\xfbt 2014 inclus sur une grande s\xe9lection de produits exp\xe9di\xe9s et vendus par Amazon uniquement (cest-\xe0-dire hors produits vendus par des vendeurs tiers sur la plateforme marketplace du site www.amazon.fr). Les d\xe9marques appliqu\xe9es sont signal\xe9es sur les pages d\xe9taill\xe9es des produits concern\xe9s. Amazon se r\xe9serve le droit de retirer, suspendre ou modifier l'offre \xe0 tout moment. Les Conditions G\xe9n\xe9rales de Vente du site www.amazon.fr s'appliquent \xe0 cette op\xe9ration. \r\n \r\n\r\nCe message a \xe9t\xe9 envoy\xe9 \xe0 james@hillyerd.com par Amazon EU S.\xe0.r.l., RCS Luxembourg, B-101818, 5 Rue Plaetis, L-2338 Luxembourg, Grand- Duch\xe9 du Luxembourg, (\xab Amazon.fr \xbb) et Amazon Services Europe S.\xe0.r.l., RCS Luxembourg, B-93815, 5 Rue Plaetis, L-2338 Luxembourg, Grand-Duch\xe9 du Luxembourg. \r\n\r\nVeuillez noter que cet e-mail promotionnel a \xe9t\xe9 envoy\xe9 \xe0 partir d'une adresse ne pouvant recevoir d'e-mails. Si vous souhaitez nous contacter, cliquez ici: http://www.amazon.fr/gp/browse.html/ref=pe_legal/?node=548536\r\n\r\nSi vous souhaitez ne plus recevoir ce type d'e-mail de la part d'Amazon.fr et Amazon Services Europe S.\xe0.r.l., cliquez ici: http://www.amazon.fr//gp/gss/o/1h4LrjIuqSr2GGNEQYjqECpL.75WDdIp.cPuPkqCDsSefYZb1.qs3Odc149px-uGX \r\n\r\nA propos de nos conditions de vente : \r\n\r\nPour toute information concernant nos conditions de vente, consultez nos conditions g\xe9n\xe9rales de vente: http://www.amazon.fr/gp/help/customer/display.html?ie=UTF8&nodeId=548524 \r\n\r\nLes produits vendus par un vendeur Marketplace sont sujets aux conditions g\xe9n\xe9rales de ce dernier. \r\n\r\nLes informations et les prix mentionn\xe9s dans ce message peuvent faire l'objet de modifications entre l'envoi de cet e-mail et le moment o\xf9 vous visitez notre site www.amazon.fr: http://www.amazon.fr/ref=pe_FootDomain. E-mail envoy\xe9 le 01/07/14 12:29 \xe0 5h00 (GMT).\r\n")
  assert.Equal(t, msg.Header.Get("MESSAGE-ID"), "<00000146f178bcdb-2ce7f105-8f4e-4785-8ad5-ee9eeeec9a59-000000@eu-west-1.amazonses.com>")
  assert.Equal(t, msg.Header.Get("DATE"), "")
}

func Test07(t *testing.T) {
  msg := readMessage("07-no_message_id.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Pensez à proteger votre habitation ")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, msg.Header.Get("MESSAGE-ID"), "")
  assert.Equal(t, msg.Header.Get("DATE"), "Tue, 01 Jul 2014 13:55:20 +0200")
}

func Test08(t *testing.T) {
  msg := readMessage("08-no_to.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Maitrisez l'Anglais sur le bout des doigts !")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, msg.Header.Get("MESSAGE-ID"), "<5b24ea5958601eb4eaad1d3d5e5cee61c4ea6d1947ad3dfe6ab52c0f157e60b3e2e27cbe8171c238@news.sprintmotorsport.com>")
  assert.Equal(t, msg.Header.Get("DATE"), "Tue, 01 Jul 2014 07:15:28 GMT")
}

func Test09(t *testing.T) {
  msg := readMessage("09-wrong_charset_in_part_header.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Confirmez l'inscription à la newsletter")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, msg.Header.Get("MESSAGE-ID"), "<130528101156TB.13164@mssysweb07>")
  assert.Equal(t, msg.Header.Get("DATE"), "Tue, 28 May 2013 10:11:56 +0200")
}

func Test10(t *testing.T) {
  msg := readMessage("10-wrong_transfer_encoding.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Digifit Fourth of July Special")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 0)
  assert.Equal(t, msg.Header.Get("MESSAGE-ID"), "<7bd20a2b-878e-442a-97b3-97040f7438cb@xtnvmta410.xt.local>")
  assert.Equal(t, msg.Header.Get("DATE"), "Thu, 03 Jul 2014 15:31:30 -0600")
}

func Test11(t *testing.T) {
  msg := readMessage("11-interleaved_text_parts.eml")
  mime, err := ParseMIMEBody(msg)
  if err != nil {
    t.Fatalf("Failed to parse non-MIME: %v", err)
  }
  _ = mime
  assert.Equal(t, mime.GetHeader("Subject"), "Test")
  assert.Equal(t, len(mime.Attachments), 0)
  assert.Equal(t, len(mime.Inlines), 2)
  assert.Equal(t, mime.Inlines[0].FileName(), "image-1.jpeg")
  assert.Equal(t, mime.Inlines[1].FileName(), "image-2.jpeg")
  assert.Equal(t, msg.Header.Get("MESSAGE-ID"), "<26627E5E-DB8B-4568-8341-52C187D5E5F2@hillyerd.com>")
  assert.Equal(t, msg.Header.Get("DATE"), "Fri, 4 Jul 2014 11:19:07 +0200")
  assert.Equal(t, mime.Html, "")
  assert.Equal(t, mime.Text, "Grunt\r\n\r\n\n--\n\r\n\r\nJames Hillyerd\r\n\r\nYoupi\r\n\r\n\n--\n\r\n\r\n\r\nEndGrunt")
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
