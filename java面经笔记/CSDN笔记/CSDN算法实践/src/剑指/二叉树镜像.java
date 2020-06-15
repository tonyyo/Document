package 剑指;

import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;

class TreeNode{
    int val = 0;
    TreeNode left = null;
    TreeNode right = null;
    TreeNode(int val){
        this.val = val;
    }
}

public class 二叉树镜像 {
    public static void main(String[] args) {
        List<Integer> list1 = new ArrayList<Integer>();
        list1.add(8);
        list1.add(6);
        list1.add(10);
        list1.add(5);
        list1.add(7);
        list1.add(9);
        list1.add(11);
        TreeNode root = buildTree(list1, 0);
        ArrayList<Integer> list = bfs(root);  // 打印原树
        for (Integer i :
                list) {
            System.out.print(String.valueOf(i) + " ");
        }
        System.out.println();
        TreeNode resultRoot = Mirror(root);
        ArrayList<Integer> list2 = bfs(resultRoot);  // 打印镜像翻转后的树
        for (Integer i :
                list2) {
            System.out.print(String.valueOf(i) + " ");
        }
    }

    static TreeNode Mirror(TreeNode root){  // 将以root为根节点的树镜像翻转
        if(root == null)
            return null;
        TreeNode tempLeft = Mirror(root.left);
        TreeNode tempRight = Mirror(root.right);
        root.left = tempRight;
        root.right = tempLeft;
        return root;
    }

    static TreeNode buildTree(List<Integer> list, int start){ // 构建二叉树
        if(start >= list.size()){
            return null;
        }
        TreeNode root = new TreeNode(list.get(start));
        root.left = buildTree(list, start * 2 + 1);
        root.right = buildTree(list, start * 2 + 2);
        return root;
    }

    static ArrayList<Integer> bfs(TreeNode root){   // 层次遍历二叉树
        ArrayList<Integer> list = new ArrayList<>();
        LinkedList<TreeNode> stack = new LinkedList<>();
        stack.add(root);
        while(stack.size() > 0){
            TreeNode pos = stack.poll();
            list.add(pos.val);
            if(pos.left != null)
                stack.add(pos.left);
            if(pos.right != null)
                stack.add(pos.right);
        }
        return list;
    }
}
