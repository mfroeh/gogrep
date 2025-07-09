# POSIX ERE regex engine
This is just a toy project for better understanding regex.\
The module implements a regex library, which uses a NFA like data structure with backtracking and a small binary in `gogrep` on top of it, which implements some of grep/ripgrep like functionality.


There is one major bug, which would require a full redesign: there is no possible backtracking, back to within a capture group, after it has been matched.\
Due to the greedy by default nature of regex, this means that when quantifiers are used at the last element within a capture group, they will always be fully exhausted.\
For example:\
`([a-Z[:ascii:]]+)\s+` will not match `"something "`, since the `[:ascii:]` will also match the space, and we can't backtrack, to have the capture group match until just before the whitespace ` `.
