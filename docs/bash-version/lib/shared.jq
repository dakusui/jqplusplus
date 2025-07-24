def scalars_or_empty:
  select(. == null or . == true or . == false or type == "number" or type == "string" or ((type=="array" or type=="object") and length==0));

def path2pexp($v):
  $v | reduce .[] as $segment (""; . + ($segment | if type == "string" then ".\"" + . + "\"" else "[\(.)]" end));
