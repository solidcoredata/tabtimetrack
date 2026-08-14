# Tab Time Track

Install:
```
go install github.com/solidcoredata/tabtimetrack@latest
```

## Purpose

Keep track of time worked and perform simple reporting.

Example File (`customer_YYYY-MM.txt`):
```
Customer Name

@rate 100.10
@breakout [856] Description of item to bill in own line-item

2023-08-01	08:00	09:00:34	[100] Added two number together. [101] Sent email.
2023-08-01	09:00:35	10:00	[856] Created the parser.
2023-08-02	10:00	14:32	Wrote documentation.
```

Process with:
```
tabtimetrack -f customer_YYYY-MM.txt -desc ab
```

## Use

Maintain a file per customer per billing cycle. Once billed, archive the file. Begin a new file for the new billing period.

In whatever text editor you typically use, configure a macro to enter in the current date and time (`date<tab>time<tab>`), and a macro to just enter in the time (`time<tab>`). 

## Capitalized expenses

With the `-cap` flag enabled, a bracketed `c` marker in a description marks that
sentence as capitalized work, much like an issue reference (`[1234]`). The line's
time is split evenly per sentence between capitalized and non-capitalized sums,
reported as `Sum-Capitalized` and `Sum-NonCapitalized`:

```
2023-08-01	08:00	10:00	[c] [856] Installed server rack. Fixed software bug.
```

The above 2 hour line reports 1 hour capitalized, 1 hour non-capitalized. The
marker may be combined with an issue reference (`[c] [856] ...` or `[856] [c] ...`)
and is case-insensitive. Use `-desc c,nc` (or `capitalized,non-capitalized`) to
print the description summaries split by capitalized vs non-capitalized.


