# The kongkow message style

Voice for every line written for a person. Help text follows the
[command standard](command-standard.md).

## Voice

Calm and plain, never blame. Say what is true about the input or the world.

| Never | Write |
|---|---|
| invalid X | X is not a … / kongkow cannot read X |
| failed to X | could not X |
| you must Y | Y first / kongkow needs Y |
| please, simply, sorry, oops, ! | (leave out) |

## Errors

- `error.message`: what happened and the state.
- `error.hint`: what to try next, often naming a command.
- Text mode: `Error:` / `Stopped:` then `Hint:` then `(code …, exit …)`.

## Success

Short and factual: `added hello`, `published hello → local`. Empty lists say
how to create the first item.
