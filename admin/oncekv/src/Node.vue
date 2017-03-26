<template>
  <div class="node">
    <el-card class="box-card">
        <div slot="header" class="clearfix">
          <span style="line-height: 36px;"> 
            <el-tag :type="tagClass">{{tag}}</el-tag>
            <el-tag v-if="isLeader" type="warning">Leader</el-tag>
            {{ host }}
          </span>
        </div>
      <div v-for="(value, key) in stats" class="text item">
        {{ capitalize(key) }}: {{ value }}
      </div>
    </el-card>
  </div>
</template>

<script>

export default {
  name: 'node',
  data () {
    return {
      stats: {}      
    }
  },
  created () {
    console.log("create")
    this.getStats()
  },
  props: {
    url: String,
    tag: String
  },
  components: {
  },
  computed: {
    tagClass () {
      return this.tag == 'cache' ? 'success' : 'primary'
    },
    isLeader () {
      return this.stats.state == 'Leader'
    },
    statsArr () {
      return [this.stats]
    },
    host () {
      let host = this.url
      let httpProtocol = 'http://'
      let httpsProtocol = 'https://'

      if (host.substring(0, httpProtocol.length) === httpProtocol) {
        return host.substring(httpProtocol.length)
      } else if (host.substring(0, httpsProtocol.length) === httpsProtocol) {
        return host.substring(httpProtocol.length) 
      }

      return host
    },
    wsUrl () {
      let addr = this.url
      let httpProtocol = 'http://'
      let httpsProtocol = 'https://'

      if (addr.substring(0, httpProtocol.length) === httpProtocol) {
        addr = 'ws://' + addr.substring(httpProtocol.length)
      } else if (addr.substring(0, httpsProtocol.length) === httpsProtocol) {
        addr = 'wss://' + addr.substring(httpProtocol.length) 
      } else {
        addr = 'ws://' + addr
      }

      return addr + '/ws/stats'
    }
  },
  watch: {
    url (to) {
      this.getStat()
    }
  },
  methods: {
    getStats () {      
      console.log("stats", this.wsUrl)
      let self = this
      let ws = new WebSocket(this.wsUrl);
      ws.onmessage = function(msg){
        self.stats = JSON.parse(msg.data)
      } 
    },
    capitalize(str) {
      return str.split("_").map(function (w) { return w.charAt(0).toUpperCase() + w.substring(1, w.length) } ).join('')
    }
  }
}
</script>
<style lang="scss">
.node {
  width: 90%;
  margin: 0 auto;
  text-align: left;
  padding: 5%;
}

.header {
  background: #e5e9f2;
}

h4 {
  text-align: left;
  text-transform: none;
}
</style>