import assert from 'node:assert/strict'
import {createServer} from 'vite'
import vue from '@vitejs/plugin-vue'
import {createServer as createHttpServer} from 'node:http'
import {createSSRApp,h} from 'vue'
import {renderToString} from '@vue/server-renderer'
import ElementPlus,{ID_INJECTION_KEY} from 'element-plus'
import {containerFields} from '../src/utils/workloadFields.js'
const server=await createServer({configFile:false,plugins:[vue()],server:{middlewareMode:true,hmr:{server:createHttpServer()}},appType:'custom'})
try {
 const {default:SchemaFields}=await server.ssrLoadModule('/src/components/SchemaFields.vue')
 const fields=containerFields['端口与命令'].filter(field=>['command','args'].includes(field.key))
 const app=createSSRApp({render:()=>h(SchemaFields,{fields,modelValue:{command:['/bin/sh','-c'],args:['echo hello world']}})})
 app.use(ElementPlus).provide(ID_INJECTION_KEY,{prefix:123,current:0})
 const html=await renderToString(app)
 assert.equal((html.match(/class="[^"]*\bfull-row\b[^"]*"/g)||[]).length,2,'Command and arguments each occupy one full row')
 assert.equal((html.match(/<input /g)||[]).length,2,'One input per field')
 assert.equal((html.match(/<textarea/g)||[]).length,0,'Both fields are single-line inputs')
 assert.match(html,/启动命令 command/)
 assert.match(html,/启动参数 args/)
 console.log('命令和参数各占一行的组件渲染检查通过')
} finally {await server.close()}
