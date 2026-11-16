import {test} from "node:test";
import assert from "node:assert/strict";
import {inspectContract} from "./serviceContract";

test("Service contract projects nested objects and arrays with local required scopes and unique definition paths",()=>{
 const schema={type:["object","array"],required:["order"],properties:{
  order:{type:"object",required:["id"],properties:{id:{type:"string"},"a/b~c":{type:"number"}}},
  items:{type:"string"}
 },items:{type:"object",properties:{qty:{type:"integer"}}}};
 const original=JSON.stringify(schema),view=inspectContract(schema);
 assert.equal(view.rows[0].type,"object | array");
 assert.equal(view.rows.find(r=>r.path==="#/properties/order")?.required,"是");
 assert.equal(view.rows.find(r=>r.path==="#/properties/order/properties/id")?.required,"是");
 assert.equal(view.rows.find(r=>r.path==="#/items")?.required,"—");
 assert.equal(view.rows.find(r=>r.path==="#/items/properties/qty")?.required,"否");
 assert.ok(view.rows.some(r=>r.path.endsWith("a~1b~0c")));
 assert.equal(new Set(view.rows.map(r=>r.path)).size,view.rows.length);
 assert.equal(JSON.stringify(schema),original);
});
test("Contract constraints preserve zero, false, empty defaults and enum values instead of inventing validation",()=>{
 const view=inspectContract({type:"integer",minimum:0,maximum:100000,multipleOf:1,default:false,const:0,enum:[0,false,""],additionalProperties:false});
 const text=view.rows[0].constraints;
 for(const expected of ["最小值：0","最大值：100000","默认值：false","固定值：0","可选值：[0,false,\"\"]","额外属性：false"])assert.ok(text.includes(expected),expected);
 assert.equal(inspectContract({default:""}).rows[0].type,"类型未声明");
 assert.ok(inspectContract({default:""}).rows[0].constraints.includes('默认值：""'));
});
test("References and composite constraints remain explicit and are never fetched or mistaken for a flat contract",()=>{
 const view=inspectContract({$ref:"https://example.invalid/schema",allOf:[{required:["id"]}],properties:{value:{$ref:"#/$defs/Value"}}});
 assert.equal(view.rows[0].type,"类型未声明");
 assert.match(view.rows[0].constraints,/allOf/);assert.match(view.rows[0].constraints,/https:\/\/example.invalid/);
 assert.ok(view.notices.some(n=>n.includes("不自动展开")));assert.ok(view.notices.some(n=>n.includes("组合")));
 assert.ok(view.raw.includes('"required"'));assert.equal(view.rows.length,2);
});
test("Missing, boolean and malformed definitions are reported; generic object output never fabricates fields",()=>{
 assert.equal(inspectContract(undefined).rows.length,0);assert.match(inspectContract(undefined).notices.join(),/未提供/);
 assert.equal(inspectContract(false).rows[0].type,"不允许任何值");assert.equal(inspectContract(true).rows[0].type,"任意值");
 assert.equal(inspectContract("invalid").rows[0].type,"无效定义");
 const output=inspectContract({type:"object"});assert.equal(output.rows.length,1);assert.match(output.notices.join(),/未提供具名字段/);
});
test("Large, deep and cyclic definitions are bounded without mutating source data",()=>{
 const properties=Object.fromEntries(Array.from({length:500},(_,i)=>["field"+i,{type:"string"}]));
 const view=inspectContract({type:"object",properties});assert.equal(view.rows.length,200);assert.match(view.notices.join(),/200/);
 let deep:any={type:"string"};for(let i=0;i<15;i++)deep={type:"array",items:deep};
 const nested=inspectContract(deep);assert.ok(nested.rows.length<=9);assert.match(nested.notices.join(),/8 层/);
 const cycle:any={type:"object",properties:{}};cycle.properties.again=cycle;
 const cyclic=inspectContract(cycle);assert.ok(cyclic.rows.length<=2);assert.match(cyclic.notices.join(),/循环/);
 const big=inspectContract({description:"x".repeat(100000)});assert.equal(big.raw.length,64000);assert.equal(big.rawTruncated,true);
});
test("Tuple items and prefixItems remain distinct; long enums preserve the full raw definition",()=>{
 const view=inspectContract({type:"array",prefixItems:[{type:"string"},false],items:true});
 assert.deepEqual(view.rows.map(r=>r.path),["#","#/prefixItems/0","#/prefixItems/1","#/items"]);
 assert.match(inspectContract({items:[{type:"string"}]}).notices.join(),/元组/);
 const values=Array.from({length:20},(_,i)=>i),long=inspectContract({enum:values});
 assert.match(long.rows[0].constraints,/共 20 项/);assert.deepEqual(JSON.parse(long.raw).enum,values);
 const literal=inspectContract({type:"string",description:"<script>alert(1)</script>"});assert.equal(literal.rows[0].description,"<script>alert(1)</script>");
});
