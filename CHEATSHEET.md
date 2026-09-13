# parley cheatsheet

Every keybinding, grouped by what you are doing. Two things to remember first:

- press `?` inside parley to pop up this list any time.
- press `ctrl+k` for the command palette - a searchable menu of every command,
  so you never have to memorize a key.

A key only fires for the panel that currently has focus. The focused panel has
a bold border. Use `tab` / `f2` / `f3` to move focus. Row keys like `a` / `d` /
`space` only apply when a rows table (params, headers, form fields) is focused.

## focus and navigation

| key | action |
| --- | --- |
| tab / shift+tab | move focus to next / previous panel |
| f2 | jump to the Request panel |
| f3 | jump to the Response panel |
| f10 | jump to the Environment rail |
| shift+left / shift+right | previous / next tab within a panel |
| esc | go back, or close a modal |

## screens

| key | screen |
| --- | --- |
| f4 | Workspaces |
| f5 | Dashboard (run history) |
| f9 | Settings |
| ctrl+e | Environments |
| ctrl+k | command palette (search every command) |
| ? | help overlay (shows all keys) |

## send and save

| key | action |
| --- | --- |
| ctrl+j or ctrl+r | send the request |
| ctrl+s | save |
| ctrl+t | cycle Body / Headers / Cookies view |
| ctrl+f | search the response |
| ctrl+y | history (recent requests) |
| ctrl+x | save example |
| ctrl+u | reveal / hide masked secrets |

## edit rows (params, headers, form fields)

| key | action |
| --- | --- |
| a | add a row |
| d | delete the current row |
| space | enable / disable the row |
| enter | edit the row |

## import, export, tools

| key | action |
| --- | --- |
| ctrl+l | import (curl, Postman v2.1, OpenAPI 3) |
| ctrl+o | export |
| ctrl+n | generate a code snippet (curl or Go) |
| ctrl+w | variables |

## layout

| key | action |
| --- | --- |
| f1 | toggle the side rail |
| f6 | toggle orientation (side by side vs stacked) |
| f7 | hide / show the request zone |
| f8 | hide / show the response zone |
| ctrl+z | zoom (maximize) the response |
| ctrl+\ | collections drawer |

## quit

| key | action |
| --- | --- |
| ctrl+c | force quit |

## masking secrets

Secret-looking values (Authorization, Set-Cookie, api keys, cookie values) are
shown as `****` by default so they stay hidden during a screen-share. Open the
Headers or Cookies tab (cycle with `ctrl+t`) and press `ctrl+u` to reveal them,
`ctrl+u` again to hide.
