# go-output v2 Migration Quick Reference

## Import Changes
| v1 | v2 |
|---|---|
| github.com/ArjenSchwarz/go-output/format | github.com/ArjenSchwarz/go-output/v2 |

## Common Replacements
| v1 | v2 |
|---|---|
| &format.OutputArray{} | output.New() |
| output.AddContents(data) | .Table("", data) |
| output.Write() | output.NewOutput().Render(ctx, doc) |
| output.Keys = [...] | output.WithKeys(...) |
| output.AddHeader("text") | .Header("text") |
| output.AddToBuffer() | (not needed) |

## Settings to Options
| v1 Setting | v2 Option |
|---|---|
| settings.OutputFormat = "json" | output.WithFormat(output.JSON) |
| settings.OutputFile = "file.ext" | output.WithWriter(output.NewFileWriter(".", "file.ext")) |
| settings.UseEmoji = true | output.WithTransformer(&output.EmojiTransformer{}) |
| settings.UseColors = true | output.WithTransformer(&output.ColorTransformer{}) |
| settings.SortKey = "Name" | output.WithTransformer(&output.SortTransformer{Key: "Name"}) |
| settings.TableStyle = "style" | output.WithTableStyle("style") |
| settings.HasTOC = true | output.WithTOC(true) |

## Progress Changes
| v1 | v2 |
|---|---|
| format.NewProgress(settings) | output.NewProgress(format, options...) |
| format.ProgressColorGreen | output.ProgressColorGreen |

## Key Concepts
1. **Context Required**: All rendering requires context.Context
2. **Builder Pattern**: Build documents, then render them
3. **No Global State**: Each instance is independent
4. **Functional Options**: Configure with WithXxx() functions
5. **Type Safety**: Compile-time format checking