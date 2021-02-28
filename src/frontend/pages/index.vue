<template>
  <v-row justify="center" align="center">
    <v-col cols="12" sm="8" md="6">
      <template v-for="image in images">
        <v-card>
          <v-card-title class="headline">
            {{ image.Title }}
          </v-card-title>
          <v-card-subtitle>
            {{ image.Date }}
          </v-card-subtitle>
          <v-card-text>
            <v-img :src="image.URL" height="480px"></v-img>
          </v-card-text>
        </v-card>
      </template>
    </v-col>
  </v-row>
</template>

<script>
export default {
  async asyncData({ app }) {
    console.log('asyncData')
    const response = await app.$axios.$get('/api/list')
    console.log('~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~')
    console.log(response)
    console.log('~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~')
    return {
      images: response,
    }
  },
  data() {
    return {
      images: [],
    }
  },
}
</script>
