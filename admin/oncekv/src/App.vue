<template>
  <div id="app">
    <h2>oncekv admin</h2>

    <el-row :gutter="10">
      <el-col v-for="url in cachesList" :key="url" :span="cacheColSpan">
        <node :url="url" :tag="cacheType"></node>
      </el-col>
    </el-row>

    <el-row :gutter="10">
      <el-col v-for="url in dbsList" :key="url" :span="dbColSpan">
        <node :url="url" :tag="dbType"></node>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import Node from './Node.vue'

console.log("env:" + process.env.NODE_ENV)

let loc = window.location, new_uri;  
let host = process.env.ONCEKV_ADMIN_ADDR
let cachesUrl = 'ws://'+host+'/ws/caches' 
let dbsUrl = 'ws://'+host+'/ws/dbs' 
let cachesWS = new WebSocket(cachesUrl);
let dbsWS = new WebSocket(dbsUrl);

export default {
  name: 'app',
  components: {
    Node
  },
  data () {
    return {
      cacheType: "cache",
      dbType: "database",
      cachesList: [],
      dbsList: [],
    }
  },
  created () {
    this.fetchUrlList()
  },
  computed: {
    cacheColSpan() {
      return this.cachesList.length > 3 ? 3 : 24/this.cachesList.length
    },
    dbColSpan() {
      return this.dbsList.length > 3 ? 3 : 24/this.dbsList.length
    }
  },
  methods: {
    fetchUrlList () {
      console.log("msg.data")
      let self = this
      cachesWS.onmessage = function(msg){
        console.log(msg.data)
        self.cachesList = JSON.parse(msg.data)
      }

      dbsWS.onmessage = function(msg){
        self.dbsList = JSON.parse(msg.data)
      }
    }
  }
}
</script>

<style lang="scss">
#app {
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin: 0px auto;
  padding-top: 20px;
  width: 65%;
  background-color: #E5E9F2;
}

h1, h2 {
  font-weight: normal;
}

ul {
  list-style-type: none;
  padding: 0;
}

li {
  display: inline-block;
  margin: 0 10px;
}

a {
  color: #42b983;
}
</style>
