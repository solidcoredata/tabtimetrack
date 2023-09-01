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


