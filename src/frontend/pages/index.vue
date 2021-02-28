<template>
  <v-row justify="center" align="center" no-gutters>
    <v-col cols="12">
      <v-btn color="pink" fab class="ma-2" @click="showNewDialog()"
        ><v-icon>mdi-plus</v-icon></v-btn
      >
    </v-col>
    <v-dialog v-model="newDialog" max-width="480px">
      <v-form>
        <v-card>
          <v-card-title
            >New Image<v-btn absolute top right @click="saveImage()"
              >SAVE</v-btn
            ></v-card-title
          >
          <v-card-text>
            <v-text-field v-model="name" label="insert name"></v-text-field>
            <v-file-input
              v-model="imageFile"
              label="select image"
            ></v-file-input>
          </v-card-text>
        </v-card>
      </v-form>
    </v-dialog>
    <v-col cols="12" xs="12" sm="8" md="6" lg="4" xl="4">
      <template v-for="image in images">
        <v-card class="pa-2 ma-2" outlined tile>
          <v-img :src="image.path"> </v-img>
          <v-card-title v-text="image.date"></v-card-title>
          <v-card-text v-text="image.name"></v-card-text>
          <v-card-actions>
            <v-btn
              disabled="disabled"
              color="grey"
              absolute
              bottom
              right
              fab
              class="mb-12"
              @click="deleteImage(image.id)"
              ><v-icon>mdi-minus</v-icon></v-btn
            >
          </v-card-actions>
        </v-card>
      </template>
    </v-col>
  </v-row>
</template>

<script>
export default {
  async asyncData({ app }) {
    const response = await app.$axios.$get('/api/list')
    return {
      images: response,
    }
  },
  data() {
    return {
      images: [],
      newDialog: false,
      editDialog: false,
      name: '',
      imageFile: null,
    }
  },
  methods: {
    showNewDialog() {
      this.newDialog = true
    },
    async saveImage() {
      const formData = new FormData()
      formData.append('name', this.name)
      formData.append('imageFile', this.imageFile)
      await this.$axios.post('/api/addImage', formData)
      this.newDialog = false
    },
    deleteImage(id) {
      console.log(id)
    },
  },
}
</script>
