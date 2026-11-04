// json-bigint ships no type declarations. The request layer only needs its
// default export (`JSONBig(options)`), and its return value is treated as
// `unknown`/`any` at the parse boundary, so a bare module declaration is enough.
declare module 'json-bigint'
