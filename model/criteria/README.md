### Criteria

Criteria provide convenient expression for conditional logic

The basic expression syntax:
   
   _criterion_: leftOperand operator rightOperator
   
   _operators:_
  - =, !=, >=, <=, <, >
  - :  [assertly](https://github.com/viant/assertly#validation) operator for contains, RegExpr, ranges etc...
    
   _predicate_: criterion [logical operator criterion]
   
   _logical operators:_
   -  &&
   -  ||



predicate examples :
```text
    $key1 = '123' || key2 > 3
```

```text
    ($key1 = '123' || key2 > 3) AND $key5:/abc/
```


Expression with $ is expanded before evaluation if corresponding path exists.  