<template>
  <div>
    <h1>石秀涛憨憨</h1>
    <div v-for="item in postList" :key="item.id">
      <h3>{{ item.title }}</h3>
      <p>{{ item.content }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
// 导入写好的接口方法
import { getPostList } from './api/post'

const postList = ref([])

// 页面加载自动请求后端接口
getPostList().then(res => {
  postList.value = res
})
</script>
