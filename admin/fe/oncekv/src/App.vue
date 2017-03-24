<template>
  <div id="app">
        <datepicker placeholder="European Format ('d-m-Y')" :config="{ dateFormat: 'd-m-Y', static: true }"></datepicker>
  </div>
</template>

<script>
import Datepicker from 'vue-bulma-datepicker'

export default {
  name: 'app',
  components: {
    Datepicker
  },
  data () {
    return {
      msg: 'Welcome to Your Vue.js App'
    }
  }
}
</script>

<style>
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
