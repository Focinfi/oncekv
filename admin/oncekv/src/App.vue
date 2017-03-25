<template>
  <div id="app">
    <p>{{ cachesList }}</p>
    <p>{{ dbsList }}</p>
    <node v-for="url in cachesList" :key="url" :url="url"></node>
  </div>
</template>

<script>
import Node from './Node.vue'

let loc = window.location, new_uri;  
let cachesUrl = 'ws://127.0.0.1:5546/ws/caches' 
let dbsUrl = 'ws://127.0.0.1:5546/ws/dbs' 

let fetchUrlList = function(self) {
  console.log(self.cachesList)
  let cachesWS = new WebSocket(cachesUrl);
  cachesWS.onmessage = function(msg){
    self.cachesList = JSON.parse(msg.data)
  }

  let dbsWS = new WebSocket(dbsUrl);
  dbsWS.onmessage = function(msg){
    self.dbsList = JSON.parse(msg.data)
  }
}

export default {
  name: 'app',
  components: {
    Node
  },
  data () {
    return {
      cachesList: [],
      dbsList: [],
    }
  },
  created () {
    this.fetchUrlList()
  },
  methods: {
    fetchUrlList () {
      console.log("msg.data")
      let self = this
      let cachesWS = new WebSocket(cachesUrl);
      cachesWS.onmessage = function(msg){
        console.log(msg.data)
        self.cachesList = JSON.parse(msg.data)
      }

      let dbsWS = new WebSocket(dbsUrl);
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
  margin-top: 60px;
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
