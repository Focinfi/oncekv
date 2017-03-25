<template>
  <div class="node">
    <p>Node: {{ wsUrl }}</p>
    <p>Stats: {{ stats }}</p>
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
    url: String
  },
  computed: {
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
    }
  }
}
</script>
