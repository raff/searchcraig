# searchcraigs
Search local classifieds on craigslist

Usage:

    searchcraigs [options...] items
    
 Where options are:
 
    -browse
    	Create HTML page and open browser
    -cat string
    	Category (default "sss")
        Use craigslist category values or all,bikes,boats,cars,phones,computers,electronics,free,music,rvs,sports,tools
    -dedup
    	Bundle duplicates (default true)
    -filter string
    	Title filter
        Multiple filter words can use booleans (one|two means title contains `one` or `two, one&two or one,two means title contains `one` and `two`)
        Filters can be negated using !word (!one means title should not contain the word `one`)
    -html
    	Return an HTML page
    -max int
    	Max price
    -min int
    	Min price
    -pictures
    	Has pictures (default true)
    -region string
    	Region (default "sfbay")
    -sort string
    	Sort type (priceasc,pricedsc,date,rel
    -subregion string
    	Subregion
    -titles
    	Search in title only
    -today
    	Added today

For example:

    searchcraigs -browse -cat=free record player
    
Search for a free record player (and show results in a browser)
